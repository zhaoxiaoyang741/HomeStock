package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultRetries  = 3
	retryBackoffSec = 1
)

// HTTPEndpoint sends events to a single URL via HTTP POST with retry logic.
type HTTPEndpoint struct {
	name       string
	url        string
	client     *http.Client
	maxRetries int
}

// NewHTTPEndpoint creates an HTTPEndpoint with the given name and URL.
func NewHTTPEndpoint(name, url string) *HTTPEndpoint {
	return &HTTPEndpoint{
		name: name,
		url:  url,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		maxRetries: defaultRetries,
	}
}

func (e *HTTPEndpoint) Name() string { return e.name }

func (e *HTTPEndpoint) Send(ctx context.Context, event OutboundEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	var lastErr error
	for i := range e.maxRetries + 1 {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoffSec * time.Second << (i - 1)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
