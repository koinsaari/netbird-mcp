package tools

import (
	"errors"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

// toolMessage returns the error text plus a hint for known API statuses.
// Client errors are safe to show because the token is redacted and URLs are dropped.
func toolMessage(err error) string {
	msg := err.Error()
	if apiErr, ok := errors.AsType[*netbird.APIError](err); ok {
		if hint := apiHint(apiErr.Status); hint != "" {
			msg += " (hint: " + hint + ")"
		}
	}
	return msg
}

func apiHint(status int) string {
	switch status {
	case 401:
		return "token invalid or expired"
	case 403:
		return "token lacks permission"
	case 400, 422:
		return "check the arguments"
	case 301, 302, 303, 307, 308:
		return "server redirected; check NETBIRD_API_URL"
	case 429:
		return "rate limited, retry later"
	}
	if status >= 500 {
		return "NetBird server error, retry later"
	}
	return ""
}
