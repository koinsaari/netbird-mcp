package netbird

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

const maxMessageLen = 512

var patFingerprint = regexp.MustCompile(`(?i)\bpat:\s*\S+`)

var ErrInvalidToken = errors.New("invalid token")

type APIError struct {
	Status  int
	Message string
	Err     error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("netbird api: %d: %s", e.Status, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func isInvalidTokenMessage(msg string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(msg)), "invalid token")
}

func newAPIError(resp *http.Response, token string) *APIError {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.UnmarshalRead(io.LimitReader(resp.Body, maxErrorBodySize), &body); err != nil {
		body.Message = ""
	}

	msg := sanitize(body.Message, token)
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	apiErr := &APIError{Status: resp.StatusCode, Message: msg}
	if isInvalidTokenMessage(body.Message) {
		apiErr.Err = ErrInvalidToken
	}
	return apiErr
}

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
	msg = patFingerprint.ReplaceAllString(msg, "pat: [redacted]")
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
