// Package netbird is a thin client for the NetBird management REST API.
package netbird

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

const (
	defaultMaxBodySize = 32 << 20
	maxErrorBodySize   = 64 << 10
)

// Client calls the NetBird management API. The token is only ever sent in
// the Authorization header and never appears in errors.
type Client struct {
	baseURL     string
	token       string
	hc          *http.Client
	maxBodySize int64
}

// New returns a client for the API at baseURL (e.g. https://api.netbird.io/api).
// hc is used as is; the caller configures timeouts and redirects.
func New(baseURL, token string, hc *http.Client) *Client {
	return &Client{baseURL: baseURL, token: token, hc: hc, maxBodySize: defaultMaxBodySize}
}

// get GETs baseURL/segments... and decodes the JSON response into out.
// Segments are escaped so caller-supplied IDs cannot change the path.
func (c *Client) get(ctx context.Context, out any, query url.Values, segments ...string) error {
	escaped := make([]string, len(segments))
	for i, seg := range segments {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("invalid path segment %q", seg)
		}
		escaped[i] = url.PathEscape(seg)
	}
	u, err := url.JoinPath(c.baseURL, escaped...)
	if err != nil {
		return fmt.Errorf("build url: %w", err)
	}
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newAPIError(resp, c.token)
	}

	body := http.MaxBytesReader(nil, resp.Body, c.maxBodySize)
	if err := json.UnmarshalRead(body, out); err != nil {
		return c.decodeError(err)
	}
	return nil
}

// decodeError gives JSON errors fixed text: decoder messages can quote the response.
func (c *Client) decodeError(err error) error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return fmt.Errorf("response too large (over %d bytes)", c.maxBodySize)
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, io.ErrUnexpectedEOF} {
		if errors.Is(err, cause) {
			return fmt.Errorf("read response: %w", cause)
		}
	}
	if netErr, ok := errors.AsType[net.Error](err); ok {
		return fmt.Errorf("read response: %w", netErr)
	}
	if semErr, ok := errors.AsType[*json.SemanticError](err); ok {
		return fmt.Errorf("decode response: cannot decode JSON %s into %v", semErr.JSONKind, semErr.GoType)
	}
	return errors.New("decode response: invalid JSON")
}
