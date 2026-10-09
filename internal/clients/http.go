// Package clients calls orders-api's downstream services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StatusError is a non-2xx answer from a downstream service.
type StatusError struct {
	Service string
	Status  int
	Body    string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: HTTP %d: %s", e.Service, e.Status, e.Body)
}

type jsonClient struct {
	service string
	base    string
	hc      *http.Client
}

func newJSONClient(service, base string) jsonClient {
	return jsonClient{service: service, base: base, hc: &http.Client{Timeout: 5 * time.Second}}
}

// do sends a JSON request and decodes a JSON answer into out (when non-nil).
func (c jsonClient) do(ctx context.Context, method, path string, body, out any, headers map[string]string) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", c.service, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return &StatusError{Service: c.service, Status: resp.StatusCode, Body: string(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
