package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	accountsURL   string
	managementURL string
	token         string
	http          *http.Client
}

func New(accountsURL, managementURL, token string) *Client {
	return &Client{
		accountsURL:   accountsURL,
		managementURL: managementURL,
		token:         token,
		http:          &http.Client{Timeout: 30 * time.Second},
	}
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is a 404 response from the API.
func IsNotFound(err error) bool {
	var e *apiError
	return err != nil && errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}

func (c *Client) doRequest(ctx context.Context, method, url string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// post targets the accounts service.
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.doRequest(ctx, http.MethodPost, c.accountsURL+path, body, out)
}

// managementPost targets the management API.
func (c *Client) managementPost(ctx context.Context, path string, body, out any) error {
	return c.doRequest(ctx, http.MethodPost, c.managementURL+path, body, out)
}

// managementGet targets the management API with GET.
func (c *Client) managementGet(ctx context.Context, path string, out any) error {
	return c.doRequest(ctx, http.MethodGet, c.managementURL+path, nil, out)
}
