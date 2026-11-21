package tools

import (
	"errors"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func toolMessage(err error) string {
	msg := err.Error()
	if errors.Is(err, netbird.ErrNotJSON) {
		return msg + " (hint: check NETBIRD_API_URL points at the NetBird API, usually ending in /api)"
	}
	if apiErr, ok := errors.AsType[*netbird.APIError](err); ok {
		hint := apiHint(apiErr.Status)
		if apiErr.InvalidToken() {
			hint = "token invalid or expired"
		}
		if hint != "" {
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
	case 404:
		return "not found; check the ID or NETBIRD_API_URL"
	case 429:
		return "rate limited, retry later"
	}
	if status >= 500 {
		return "NetBird server error, retry later"
	}
	return ""
}
