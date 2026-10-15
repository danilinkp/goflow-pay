package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Post(ctx context.Context, path string, body any, result any) error {
	return c.doWithHeaders(ctx, http.MethodPost, path, nil, body, result)
}

func (c *Client) PostWithHeaders(ctx context.Context, path string, headers map[string]string, body any, result any) error {
	return c.doWithHeaders(ctx, http.MethodPost, path, headers, body, result)
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.doWithHeaders(ctx, http.MethodGet, path, nil, nil, result)
}

func (c *Client) Delete(ctx context.Context, path string, result any) error {
	return c.doWithHeaders(ctx, http.MethodDelete, path, nil, nil, result)
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	return c.doWithHeaders(ctx, method, path, nil, body, result)
}

func (c *Client) doWithHeaders(ctx context.Context, method, path string, headers map[string]string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if result != nil {
		if err = json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}
