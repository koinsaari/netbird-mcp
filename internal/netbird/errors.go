package netbird

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"
)

const maxMessageLen = 512

// APIError is a non-2xx response from the NetBird API.
type APIError struct {
	Status  int
	Message string // sanitized message from the response body, or the status text
}

func (e *APIError) Error() string {
	return fmt.Sprintf("netbird api: %d: %s", e.Status, e.Message)
}

func newAPIError(resp *http.Response, token string) *APIError {
	var body struct {
		Message string `json:"message"`
	}
	// Any read or decode failure falls back to the status text.
	if err := json.UnmarshalRead(io.LimitReader(resp.Body, maxErrorBodySize), &body); err != nil {
		body.Message = ""
	}

	msg := sanitize(body.Message, token)
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return &APIError{Status: resp.StatusCode, Message: msg}
}

// sanitize strips control and invisible characters and caps the length. A
// message that contains the token in any form is dropped entirely, because
// partial redaction can be fooled by splitting the token.
func sanitize(msg, token string) string {
	if token != "" && strings.Contains(squash(msg), token) {
		return ""
	}
	msg = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			return -1
		}
		return r
	}, msg)
	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > maxMessageLen {
		msg = strings.ToValidUTF8(msg[:maxMessageLen], "")
	}
	return strings.TrimSpace(msg)
}

func squash(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
}
