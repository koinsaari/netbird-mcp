package tools

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestToolMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"invalid ip", errInvalidIP, "invalid ip: must be an IP address without zone"},
		{"401", &netbird.APIError{Status: 401, Message: "x"}, "netbird api: 401: x (hint: token invalid or expired)"},
		{"403", &netbird.APIError{Status: 403, Message: "x"}, "netbird api: 403: x (hint: token lacks permission)"},
		{"400", &netbird.APIError{Status: 400, Message: "bad ip"}, "netbird api: 400: bad ip (hint: check the filter arguments)"},
		{"422", &netbird.APIError{Status: 422, Message: "bad"}, "netbird api: 422: bad (hint: check the filter arguments)"},
		{"429", &netbird.APIError{Status: 429, Message: "x"}, "netbird api: 429: x (hint: rate limited, retry later)"},
		{"503", &netbird.APIError{Status: 503, Message: "down"}, "netbird api: 503: down (hint: NetBird server error, retry later)"},
		{"404 no hint", &netbird.APIError{Status: 404, Message: "nope"}, "netbird api: 404: nope"},
		{"wrapped 400", fmt.Errorf("list peers: %w", &netbird.APIError{Status: 400, Message: "bad ip"}), "list peers: netbird api: 400: bad ip (hint: check the filter arguments)"},
		{"canceled", fmt.Errorf("list peers: %w", context.Canceled), "list peers: context canceled"},
		{"plain", errors.New("boom"), "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := toolMessage(tt.err); got != tt.want {
				t.Errorf("toolMessage = %q, want %q", got, tt.want)
			}
		})
	}
}
