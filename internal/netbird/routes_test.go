package netbird

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListRoutesRequest(t *testing.T) {
	t.Parallel()
	var got *http.Request
	c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte("[]"))
	})
	if _, err := c.ListRoutes(t.Context()); err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.Method)
	}
	if got.URL.Path != "/api/routes" {
		t.Errorf("path = %q, want /api/routes", got.URL.Path)
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
}

func TestListRoutesDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Route
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []Route{}},
		{
			name: "full network route with unknown fields",
			body: `[{"id":"r1","network_type":"IPv4","description":"office","network_id":"office-net","enabled":true,
				"peer":"p1","peer_groups":["pg1","pg2"],"network":"10.0.0.0/24","domains":["a.example.com"],
				"metric":150,"masquerade":true,"groups":["g1"],"keep_route":true,
				"access_control_groups":["acg1"],"skip_auto_apply":true,"future_field":{"x":1}}]`,
			want: []Route{{
				ID: "r1", NetworkType: "IPv4", Description: "office", NetworkID: "office-net", Enabled: true,
				Peer: "p1", PeerGroups: []string{"pg1", "pg2"}, Network: "10.0.0.0/24",
				Domains: []string{"a.example.com"}, Metric: 150, Masquerade: true, Groups: []string{"g1"},
				KeepRoute: true, AccessControlGroups: []string{"acg1"}, SkipAutoApply: true,
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"r2","network_type":"Domain","network_id":"dom","enabled":false,"metric":9999,
				"masquerade":false,"groups":[],"keep_route":false}]`,
			want: []Route{{
				ID: "r2", NetworkType: "Domain", NetworkID: "dom", Metric: 9999, Groups: []string{},
			}},
		},
		{
			name: "order kept",
			body: `[{"id":"b"},{"id":"a"}]`,
			want: []Route{{ID: "b"}, {ID: "a"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.ListRoutes(t.Context())
			if err != nil {
				t.Fatalf("ListRoutes: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListRoutesAPIError(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{name: "json message", status: 403, body: `{"message":"no permission"}`, wantMessage: "no permission"},
		{name: "token echoed", status: 400, body: `{"message":"bad token nbp_SENTINEL here"}`, wantMessage: "Bad Request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := c.ListRoutes(t.Context())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *APIError", err)
			}
			if apiErr.Status != tt.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, tt.status)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if !strings.HasPrefix(err.Error(), "list routes: ") {
				t.Errorf("err = %q, want list routes prefix", err)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}
