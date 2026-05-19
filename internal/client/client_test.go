package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil → false", nil, false},
		{"non-API error → false", errors.New("network blip"), false},
		{"400 → false", &apiError{StatusCode: 400, Body: "bad request"}, false},
		{"404 → true", &apiError{StatusCode: 404, Body: "not found"}, true},
		{"500 → false", &apiError{StatusCode: 500, Body: "boom"}, false},
		{"wrapped 404 → true (errors.As traversal)",
			wrap("instance lookup:", &apiError{StatusCode: 404}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// wrap returns an error that wraps inner with fmt.Errorf("%w") semantics.
func wrap(prefix string, inner error) error {
	return &wrappedErr{prefix: prefix, inner: inner}
}

type wrappedErr struct {
	prefix string
	inner  error
}

func (w *wrappedErr) Error() string { return w.prefix + " " + w.inner.Error() }
func (w *wrappedErr) Unwrap() error { return w.inner }

// ── managementPostData / managementGetData: envelope unwrapping ───────────────

func TestManagementPostData_UnwrapsDataEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "abc-123", "name": "alpha"}}`))
	}))
	defer srv.Close()

	c := New("unused-accounts-url", srv.URL, "test-token")

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.managementPostData(context.Background(), "/api/v1/test", map[string]string{"foo": "bar"}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "abc-123" {
		t.Errorf("ID = %q, want %q (envelope not unwrapped)", out.ID, "abc-123")
	}
	if out.Name != "alpha" {
		t.Errorf("Name = %q, want %q", out.Name, "alpha")
	}
}

func TestManagementGetData_UnwrapsDataEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"regions": [{"name": "eu", "deployable": true}]}}`))
	}))
	defer srv.Close()

	c := New("unused", srv.URL, "tok")

	var out struct {
		Regions []struct {
			Name       string `json:"name"`
			Deployable bool   `json:"deployable"`
		} `json:"regions"`
	}
	if err := c.managementGetData(context.Background(), "/api/v1/regions", &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Regions) != 1 || out.Regions[0].Name != "eu" {
		t.Errorf("regions not decoded correctly: %+v", out.Regions)
	}
}

func TestManagementPostData_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","status":404}}`))
	}))
	defer srv.Close()

	c := New("unused", srv.URL, "tok")
	var out struct{ ID string }
	err := c.managementPostData(context.Background(), "/api/v1/missing", nil, &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound = false; expected true for 404 response. Error: %v", err)
	}
}

func TestManagementPostData_NilOutSkipsDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "x"}}`))
	}))
	defer srv.Close()

	c := New("unused", srv.URL, "tok")
	// Passing nil for `out` should be safe — caller doesn't care about the body.
	if err := c.managementPostData(context.Background(), "/api/v1/test", nil, nil); err != nil {
		t.Errorf("unexpected error with nil out: %v", err)
	}
}

func TestManagementPostData_EmptyDataField(t *testing.T) {
	// Some endpoints return {"data": null} or {"data": {}} for success-only responses.
	// We should not crash; out should be left untouched / zero.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": null}`))
	}))
	defer srv.Close()

	c := New("unused", srv.URL, "tok")
	var out struct {
		ID string `json:"id"`
	}
	if err := c.managementPostData(context.Background(), "/api/v1/noop", nil, &out); err != nil {
		t.Errorf("null data should not error, got %v", err)
	}
	if out.ID != "" {
		t.Errorf("expected out zero-valued for null data, got %+v", out)
	}
}

// ── doRequest: error body surfaces in error message ───────────────────────────

func TestDoRequest_ErrorBodyInMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"something broke"}`))
	}))
	defer srv.Close()

	c := New("unused", srv.URL, "tok")
	err := c.managementPost(context.Background(), "/api/v1/test", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
	if !strings.Contains(err.Error(), "something broke") {
		t.Errorf("error should include response body: %v", err)
	}
}
