package netbird

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestNetworkRequests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		call     func(ctx context.Context, c *Client, id string) error
		id       string
		wantPath string
		wantRaw  string
	}{
		{
			name:     "networks",
			call:     func(ctx context.Context, c *Client, _ string) error { _, err := c.ListNetworks(ctx); return err },
			wantPath: "/api/networks", wantRaw: "",
		},
		{
			name: "routers",
			call: func(ctx context.Context, c *Client, id string) error {
				_, err := c.ListNetworkRouters(ctx, id)
				return err
			},
			id: "n1", wantPath: "/api/networks/n1/routers", wantRaw: "",
		},
		{
			name: "resources",
			call: func(ctx context.Context, c *Client, id string) error {
				_, err := c.ListNetworkResources(ctx, id)
				return err
			},
			id: "n1", wantPath: "/api/networks/n1/resources", wantRaw: "",
		},
		{
			name: "routers id escaped",
			call: func(ctx context.Context, c *Client, id string) error {
				_, err := c.ListNetworkRouters(ctx, id)
				return err
			},
			id: "a/b?x=1", wantPath: "/api/networks/a/b?x=1/routers", wantRaw: "/api/networks/a%2Fb%3Fx=1/routers",
		},
		{
			name: "resources id escaped",
			call: func(ctx context.Context, c *Client, id string) error {
				_, err := c.ListNetworkResources(ctx, id)
				return err
			},
			id: "../x", wantPath: "/api/networks/../x/resources", wantRaw: "/api/networks/..%2Fx/resources",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got *http.Request
			c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
				got = r
				_, _ = w.Write([]byte("[]"))
			})
			if err := tt.call(t.Context(), c, tt.id); err != nil {
				t.Fatalf("call: %v", err)
			}
			if got.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", got.Method)
			}
			if got.URL.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", got.URL.Path, tt.wantPath)
			}
			if tt.wantRaw != "" && got.URL.EscapedPath() != tt.wantRaw {
				t.Errorf("escaped path = %q, want %q", got.URL.EscapedPath(), tt.wantRaw)
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

func serveBody(t *testing.T, body string) *Client {
	t.Helper()
	return newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}

func TestListNetworksDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []Network
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []Network{}},
		{
			name: "full with unknown fields",
			body: `[{"id":"n1","name":"office","description":"HQ","routers":["r1","r2"],"routing_peers_count":2,
				"resources":["x1"],"policies":["p1"],"future":1}]`,
			want: []Network{{
				ID: "n1", Name: "office", Description: "HQ", Routers: []string{"r1", "r2"},
				RoutingPeersCount: 2, Resources: []string{"x1"}, Policies: []string{"p1"},
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"n2","name":"lab","routing_peers_count":0}]`,
			want: []Network{{ID: "n2", Name: "lab"}},
		},
		{name: "order kept", body: `[{"id":"b"},{"id":"a"}]`, want: []Network{{ID: "b"}, {ID: "a"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListNetworks(t.Context())
			if err != nil {
				t.Fatalf("ListNetworks: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListNetworkRoutersDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []NetworkRouter
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []NetworkRouter{}},
		{
			name: "full with unknown fields",
			body: `[{"id":"r1","peer":"p1","peer_groups":["g1","g2"],"metric":100,"masquerade":true,"enabled":true,"future":1}]`,
			want: []NetworkRouter{{
				ID: "r1", Peer: "p1", PeerGroups: []string{"g1", "g2"}, Metric: 100, Masquerade: true, Enabled: true,
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"r2","peer_groups":["g1"],"metric":9999,"masquerade":false,"enabled":false}]`,
			want: []NetworkRouter{{ID: "r2", PeerGroups: []string{"g1"}, Metric: 9999}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListNetworkRouters(t.Context(), "n1")
			if err != nil {
				t.Fatalf("ListNetworkRouters: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListNetworkResourcesDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []NetworkResource
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []NetworkResource{}},
		{
			name: "full with unknown fields",
			body: `[{"id":"x1","type":"subnet","name":"lan","description":"office lan","address":"10.0.0.0/24","enabled":true,
				"groups":[{"id":"g1","name":"devs","peers_count":2,"resources_count":5,"issued":"api"}],"future":1}]`,
			want: []NetworkResource{{
				ID: "x1", Type: "subnet", Name: "lan", Description: "office lan", Address: "10.0.0.0/24", Enabled: true,
				Groups: []Group{{ID: "g1", Name: "devs", PeersCount: 2, ResourcesCount: 5, Issued: "api"}},
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"x2","type":"host","name":"db","address":"1.1.1.1","enabled":false,"groups":[]}]`,
			want: []NetworkResource{{ID: "x2", Type: "host", Name: "db", Address: "1.1.1.1", Groups: []Group{}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListNetworkResources(t.Context(), "n1")
			if err != nil {
				t.Fatalf("ListNetworkResources: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNetworkAPIError(t *testing.T) {
	t.Parallel()
	calls := []struct {
		name   string
		prefix string
		call   func(ctx context.Context, c *Client) error
	}{
		{"networks", "list networks: ", func(ctx context.Context, c *Client) error { _, err := c.ListNetworks(ctx); return err }},
		{"routers", "list network routers: ", func(ctx context.Context, c *Client) error { _, err := c.ListNetworkRouters(ctx, "n1"); return err }},
		{"resources", "list network resources: ", func(ctx context.Context, c *Client) error { _, err := c.ListNetworkResources(ctx, "n1"); return err }},
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

func TestNetworkInvalidID(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"", ".", ".."} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			called := false
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				called = true
				_, _ = w.Write([]byte("[]"))
			})
			if _, err := c.ListNetworkRouters(t.Context(), id); err == nil {
				t.Error("ListNetworkRouters: want error")
			}
			if _, err := c.ListNetworkResources(t.Context(), id); err == nil {
				t.Error("ListNetworkResources: want error")
			}
			if called {
				t.Error("request sent for invalid id")
			}
		})
	}
}
