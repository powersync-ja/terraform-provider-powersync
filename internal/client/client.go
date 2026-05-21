package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// managementEnvelope is the response wrapper used by the management API.
type managementEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type Client struct {
	accountsURL   string
	managementURL string
	// alphaURL is derived from managementURL. Once the management API exposes
	// the Alpha endpoints (apps/create, apps/delete) directly, we can drop this
	// field and route alphaPost through managementPost.
	alphaURL string
	token    string
	http     *http.Client
}

func New(accountsURL, managementURL, token string) *Client {
	return &Client{
		accountsURL:   accountsURL,
		managementURL: managementURL,
		alphaURL:      deriveAlphaURL(managementURL),
		token:         token,
		http:          &http.Client{Timeout: 30 * time.Second},
	}
}

// deriveAlphaURL swaps `powersync-api.` for `alpha.` in the management host.
// Works for both prod (powersync-api.journeyapps.com → alpha.journeyapps.com)
// and staging. No-op for non-conforming URLs, which will yield 404s — fix the
// management_url if you're on a custom host.
func deriveAlphaURL(managementURL string) string {
	return strings.Replace(managementURL, "powersync-api.", "alpha.", 1)
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

// Retry parameters. Tuned so a transient ~30-second API blip is invisible to
// the user but a sustained outage fails fast enough to be actionable.
// Declared as `var` (not `const`) so unit tests can lower them temporarily.
var (
	maxRetries     = 4                // 4 retries = 5 total attempts
	initialBackoff = 1 * time.Second  // first sleep before retry
	maxBackoff     = 16 * time.Second // upper bound per attempt
)

// isTransient reports whether err is worth retrying. Currently: 5xx API errors,
// and any error that isn't a structured apiError (i.e. network / transport /
// decode failures, which usually indicate a connection or body truncation).
// 4xx errors are NOT retried — the request itself is wrong, retrying won't help.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}
	// Non-apiError means the request never made it to a structured API response
	// (network failure, body read failure, etc.) — worth retrying.
	return true
}

// retryTransient runs fn up to maxRetries+1 times, retrying on transient errors
// with exponential backoff. Returns the last error if all attempts fail, or
// ctx.Err() if the context is cancelled mid-backoff. Use only for *idempotent*
// operations — never for create/deploy/destroy where a retry could duplicate work.
func retryTransient(ctx context.Context, fn func() error) error {
	backoff := initialBackoff
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
		err := fn()
		if err == nil {
			return nil
		}
		if !isTransient(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("after %d attempts: %w", maxRetries+1, lastErr)
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

// postData is like post but unwraps the {"data": ...} response envelope.
func (c *Client) postData(ctx context.Context, path string, body, out any) error {
	var wrapper managementEnvelope
	if err := c.post(ctx, path, body, &wrapper); err != nil {
		return err
	}
	if out != nil && len(wrapper.Data) > 0 {
		return json.Unmarshal(wrapper.Data, out)
	}
	return nil
}

// managementPost targets the management API.
func (c *Client) managementPost(ctx context.Context, path string, body, out any) error {
	return c.doRequest(ctx, http.MethodPost, c.managementURL+path, body, out)
}

// alphaPost targets the Alpha service. To be folded into managementPost when
// the management API exposes Alpha endpoints — just change the URL here.
func (c *Client) alphaPost(ctx context.Context, path string, body, out any) error {
	return c.doRequest(ctx, http.MethodPost, c.alphaURL+path, body, out)
}

// alphaPostData is alphaPost with the {"data": ...} envelope unwrapping.
func (c *Client) alphaPostData(ctx context.Context, path string, body, out any) error {
	var wrapper managementEnvelope
	if err := c.alphaPost(ctx, path, body, &wrapper); err != nil {
		return err
	}
	if out != nil && len(wrapper.Data) > 0 {
		return json.Unmarshal(wrapper.Data, out)
	}
	return nil
}

// managementGet targets the management API with GET.
func (c *Client) managementGet(ctx context.Context, path string, out any) error {
	return c.doRequest(ctx, http.MethodGet, c.managementURL+path, nil, out)
}

// managementPostData is like managementPost but unwraps the {"data": ...} response envelope.
func (c *Client) managementPostData(ctx context.Context, path string, body, out any) error {
	var wrapper managementEnvelope
	if err := c.managementPost(ctx, path, body, &wrapper); err != nil {
		return err
	}
	if out != nil && len(wrapper.Data) > 0 {
		return json.Unmarshal(wrapper.Data, out)
	}
	return nil
}

// managementGetData is like managementGet but unwraps the {"data": ...} response envelope.
func (c *Client) managementGetData(ctx context.Context, path string, out any) error {
	var wrapper managementEnvelope
	if err := c.managementGet(ctx, path, &wrapper); err != nil {
		return err
	}
	if out != nil && len(wrapper.Data) > 0 {
		return json.Unmarshal(wrapper.Data, out)
	}
	return nil
}
