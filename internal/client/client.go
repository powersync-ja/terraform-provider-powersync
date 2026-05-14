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
