// Package config loads server configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// DefaultAPIURL is the NetBird cloud management API base URL.
const DefaultAPIURL = "https://api.netbird.io/api"

// Config holds the server configuration.
type Config struct {
	APIURL string
	Token  string
}

// Load reads the configuration through getenv (os.Getenv in production).
// Errors never contain the token.
func Load(getenv func(string) string) (Config, error) {
	token := strings.TrimSpace(getenv("NETBIRD_TOKEN"))
	if token == "" {
		return Config{}, errors.New("NETBIRD_TOKEN is required")
	}

	raw := strings.TrimSpace(getenv("NETBIRD_API_URL"))
	if raw == "" {
		raw = DefaultAPIURL
	}
	apiURL, err := validateURL(raw)
	if err != nil {
		return Config{}, fmt.Errorf("invalid NETBIRD_API_URL: %w", err)
	}

	return Config{APIURL: apiURL, Token: token}, nil
}

func validateURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		// url.Parse errors echo the input, which may contain credentials.
		return "", errors.New("not a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("scheme must be http or https")
	}
	if u.User != nil {
		return "", errors.New("must not contain credentials")
	}
	if u.Host == "" {
		return "", errors.New("host is required")
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", errors.New("must not contain a query or fragment")
	}
	return strings.TrimRight(raw, "/"), nil
}
