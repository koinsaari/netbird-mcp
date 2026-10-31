package tools

import (
	"errors"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

// toolMessage returns the error text plus a hint for known API statuses.
// Client errors are safe to show because token redacted, no URL. API errors carry NetBird's message.
func toolMessage(err error) string {
	apiErr, ok := errors.AsType[*netbird.APIError](err)
	if !ok {
		return err.Error()
	}
	if hint := apiHint(apiErr.Status); hint != "" {
		return err.Error() + " (hint: " + hint + ")"
	}
	return err.Error()
}

func apiHint(status int) string {
	switch status {
	case 401:
		return "token invalid or expired"
	case 403:
		return "token lacks permission"
	case 400, 422:
		return "check the filter arguments"
	case 429:
		return "rate limited, retry later"
	}
	if status >= 500 {
		return "NetBird server error, retry later"
	}
	return ""
}
