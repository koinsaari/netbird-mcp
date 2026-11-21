package netbird

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAPIErrorRedactsPATFingerprint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		body        string
		wantMessage string
		wantInvalid bool
	}{
		{
			name:        "live invalid token body",
			body:        `{"message":"invalid token: pat: hx5yu+jgybu36zh2xxof8ulcpwxde4hognuwww4iswo= not found"}`,
			wantMessage: "invalid token: pat: [redacted] not found",
			wantInvalid: true,
		},
		{
			name:        "uppercase prefix",
			body:        `{"message":"Invalid Token: PAT:abc123== missing"}`,
			wantMessage: "Invalid Token: pat: [redacted] missing",
			wantInvalid: true,
		},
		{name: "no pat untouched", body: `{"message":"peer not found"}`, wantMessage: "peer not found"},
		{name: "raw token dropped", body: `{"message":"invalid token nbp_SENTINEL"}`, wantMessage: "Not Found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			apiErr := newAPIError(resp, sentinelToken)
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if got := apiErr.InvalidToken(); got != tt.wantInvalid {
				t.Errorf("InvalidToken = %v, want %v", got, tt.wantInvalid)
			}
			if strings.Contains(apiErr.Message, "hx5yu") {
				t.Errorf("message leaks fingerprint: %q", apiErr.Message)
			}
		})
	}
}

func TestInvalidToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		message string
		want    bool
	}{
		{"invalid token: x", true},
		{"Invalid Token", true},
		{"peer not found", false},
		{"the invalid token", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			t.Parallel()
			e := &APIError{Status: 404, Message: tt.message}
			if got := e.InvalidToken(); got != tt.want {
				t.Errorf("InvalidToken = %v, want %v", got, tt.want)
			}
		})
	}
}
