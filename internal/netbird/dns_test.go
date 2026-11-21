package netbird

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestDNSRequests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		call     func(ctx context.Context, c *Client) error
		wantPath string
	}{
		{"nameservers", "[]", func(ctx context.Context, c *Client) error { _, err := c.ListNameserverGroups(ctx); return err }, "/api/dns/nameservers"},
		{"settings", "{}", func(ctx context.Context, c *Client) error { _, err := c.GetDNSSettings(ctx); return err }, "/api/dns/settings"},
		{"zones", "[]", func(ctx context.Context, c *Client) error { _, err := c.ListDNSZones(ctx); return err }, "/api/dns/zones"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got *http.Request
			c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
				got = r
				_, _ = w.Write([]byte(tt.body))
			})
			if err := tt.call(t.Context(), c); err != nil {
				t.Fatalf("call: %v", err)
			}
			if got.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", got.Method)
			}
			if got.URL.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", got.URL.Path, tt.wantPath)
			}
			if got.URL.RawQuery != "" {
				t.Errorf("query = %q, want empty", got.URL.RawQuery)
			}
			if h := got.Header.Get("Authorization"); h != "Token "+sentinelToken {
				t.Errorf("Authorization = %q", h)
			}
			if h := got.Header.Get("Accept"); h != "application/json" {
				t.Errorf("Accept = %q", h)
			}
		})
	}
}

func TestListNameserverGroupsDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []NameserverGroup
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []NameserverGroup{}},
		{
			name: "full with unknown fields",
			body: `[{"id":"g1","name":"corp","description":"Corp DNS",
				"nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53,"x":1}],
				"enabled":true,"groups":["a","b"],"primary":false,"domains":["corp.example"],
				"search_domains_enabled":true,"future":1}]`,
			want: []NameserverGroup{{
				ID: "g1", Name: "corp", Description: "Corp DNS",
				Nameservers: []Nameserver{{IP: "1.1.1.1", NSType: "udp", Port: 53}},
				Enabled:     true, Groups: []string{"a", "b"}, Primary: false,
				Domains: []string{"corp.example"}, SearchDomainsEnabled: true,
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"g2","name":"x"}]`,
			want: []NameserverGroup{{ID: "g2", Name: "x"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListNameserverGroups(t.Context())
			if err != nil {
				t.Fatalf("ListNameserverGroups: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetDNSSettingsDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want DNSSettings
	}{
		{name: "null", body: "null", want: DNSSettings{}},
		{name: "empty object", body: "{}", want: DNSSettings{}},
		{
			name: "full with unknown fields",
			body: `{"disabled_management_groups":["g1","g2"],"future":1}`,
			want: DNSSettings{DisabledManagementGroups: []string{"g1", "g2"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).GetDNSSettings(t.Context())
			if err != nil {
				t.Fatalf("GetDNSSettings: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListDNSZonesDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []DNSZone
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []DNSZone{}},
		{
			name: "full with unknown fields",
			body: `[{"id":"z1","name":"internal","domain":"example.internal","enabled":true,
				"enable_search_domain":true,"distribution_groups":["g1"],
				"records":[{"id":"r1","name":"www.example.internal","type":"A","content":"10.0.0.1","ttl":300,"x":1}],
				"future":1}]`,
			want: []DNSZone{{
				ID: "z1", Name: "internal", Domain: "example.internal", Enabled: true,
				EnableSearchDomain: true, DistributionGroups: []string{"g1"},
				Records: []DNSRecord{{ID: "r1", Name: "www.example.internal", Type: "A", Content: "10.0.0.1", TTL: 300}},
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"z2","name":"n","domain":"d"}]`,
			want: []DNSZone{{ID: "z2", Name: "n", Domain: "d"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListDNSZones(t.Context())
			if err != nil {
				t.Fatalf("ListDNSZones: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDNSAPIError(t *testing.T) {
	t.Parallel()
	calls := []struct {
		name   string
		prefix string
		call   func(ctx context.Context, c *Client) error
	}{
		{"nameservers", "list nameserver groups: ", func(ctx context.Context, c *Client) error { _, err := c.ListNameserverGroups(ctx); return err }},
		{"settings", "get dns settings: ", func(ctx context.Context, c *Client) error { _, err := c.GetDNSSettings(ctx); return err }},
		{"zones", "list dns zones: ", func(ctx context.Context, c *Client) error { _, err := c.ListDNSZones(ctx); return err }},
	}
	errs := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{name: "json message", status: 403, body: `{"message":"no permission"}`, wantMessage: "no permission"},
		{name: "token echoed", status: 400, body: `{"message":"bad token nbp_SENTINEL here"}`, wantMessage: "Bad Request"},
	}
	for _, cl := range calls {
		for _, e := range errs {
			t.Run(cl.name+"/"+e.name, func(t *testing.T) {
				t.Parallel()
				c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(e.status)
					_, _ = w.Write([]byte(e.body))
				})
				err := cl.call(t.Context(), c)
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("err = %v, want *APIError", err)
				}
				if apiErr.Status != e.status {
					t.Errorf("Status = %d, want %d", apiErr.Status, e.status)
				}
				if apiErr.Message != e.wantMessage {
					t.Errorf("Message = %q, want %q", apiErr.Message, e.wantMessage)
				}
				if !strings.HasPrefix(err.Error(), cl.prefix) {
					t.Errorf("err = %q, want prefix %q", err, cl.prefix)
				}
				if strings.Contains(err.Error(), sentinelToken) {
					t.Errorf("err leaks token: %q", err)
				}
			})
		}
	}
}
