package config

import (
	"strings"
	"testing"
)

const sentinelToken = "nbp_SENTINEL"

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults url",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken},
			want: Config{APIURL: DefaultAPIURL, Token: sentinelToken},
		},
		{
			name: "trims token and url",
			env:  map[string]string{"NETBIRD_TOKEN": "  " + sentinelToken + "\n", "NETBIRD_API_URL": " https://nb.example.com/api "},
			want: Config{APIURL: "https://nb.example.com/api", Token: sentinelToken},
		},
		{
			name: "removes trailing slash",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https://nb.example.com/api/"},
			want: Config{APIURL: "https://nb.example.com/api", Token: sentinelToken},
		},
		{
			name: "keeps path unchanged",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "http://localhost:8080/prefix"},
			want: Config{APIURL: "http://localhost:8080/prefix", Token: sentinelToken},
		},
		{
			name:    "missing token",
			env:     map[string]string{},
			wantErr: "NETBIRD_TOKEN is required",
		},
		{
			name:    "blank token",
			env:     map[string]string{"NETBIRD_TOKEN": "   "},
			wantErr: "NETBIRD_TOKEN is required",
		},
		{
			name:    "bad scheme",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "ftp://nb.example.com/api"},
			wantErr: "scheme must be http or https",
		},
		{
			name:    "no host",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https:///api"},
			wantErr: "host is required",
		},
		{
			name: "http localhost allowed",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "http://localhost:8080/api"},
			want: Config{APIURL: "http://localhost:8080/api", Token: sentinelToken},
		},
		{
			name: "http loopback ip allowed",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "http://[::1]:8080/api"},
			want: Config{APIURL: "http://[::1]:8080/api", Token: sentinelToken},
		},
		{
			name: "http localhost any case allowed",
			env:  map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "http://LocalHost.:8080/api"},
			want: Config{APIURL: "http://LocalHost.:8080/api", Token: sentinelToken},
		},
		{
			name:    "http remote rejected",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "http://nb.example.com/api"},
			wantErr: "http is only allowed for localhost",
		},
		{
			name:    "port without host",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https://:8443/api"},
			wantErr: "host is required",
		},
		{
			name:    "unparsable",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "://bad"},
			wantErr: "invalid NETBIRD_API_URL",
		},
		{
			name:    "userinfo rejected",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https://user:pass@nb.example.com/api"},
			wantErr: "must not contain credentials",
		},
		{
			name:    "query rejected",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https://nb.example.com/api?x=1"},
			wantErr: "must not contain a query or fragment",
		},
		{
			name:    "fragment rejected",
			env:     map[string]string{"NETBIRD_TOKEN": sentinelToken, "NETBIRD_API_URL": "https://nb.example.com/api#x"},
			wantErr: "must not contain a query or fragment",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Load(func(k string) string { return tt.env[k] })
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("err = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("err = %q, want to contain %q", err, tt.wantErr)
				}
				if strings.Contains(err.Error(), sentinelToken) {
					t.Errorf("err leaks token: %q", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
