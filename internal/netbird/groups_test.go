package netbird

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListGroupsRequest(t *testing.T) {
	tests := []struct {
		name      string
		params    ListGroupsParams
		wantQuery string
	}{
		{name: "no params", wantQuery: ""},
		{name: "name", params: ListGroupsParams{Name: "dev team"}, wantQuery: "name=dev+team"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got *http.Request
			c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
				got = r
				_, _ = w.Write([]byte("[]"))
			})
			if _, err := c.ListGroups(t.Context(), tt.params); err != nil {
				t.Fatalf("ListGroups: %v", err)
			}
			if got.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", got.Method)
			}
			if got.URL.Path != "/api/groups" {
				t.Errorf("path = %q, want /api/groups", got.URL.Path)
			}
			if got.URL.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.RawQuery, tt.wantQuery)
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

func TestListGroupsDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Group
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []Group{}},
		{
			name: "full group with unknown fields",
			body: `[{"id":"g1","name":"All","peers_count":2,"resources_count":1,"issued":"api",
				"peers":[{"id":"p1","name":"laptop","future_peer_field":1},{"id":"p2","name":"phone"}],
				"resources":[{"id":"r1","type":"host","future_resource_field":true}],"future_field":{"x":1}}]`,
			want: []Group{{
				ID: "g1", Name: "All", PeersCount: 2, ResourcesCount: 1, Issued: "api",
				Peers:     []PeerMinimum{{ID: "p1", Name: "laptop"}, {ID: "p2", Name: "phone"}},
				Resources: []Resource{{ID: "r1", Type: "host"}},
			}},
		},
		{
			name: "issued omitted",
			body: `[{"id":"g2","name":"empty","peers_count":0,"resources_count":0,"peers":[],"resources":[]}]`,
			want: []Group{{ID: "g2", Name: "empty", Peers: []PeerMinimum{}, Resources: []Resource{}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.ListGroups(t.Context(), ListGroupsParams{})
			if err != nil {
				t.Fatalf("ListGroups: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListGroupsAPIError(t *testing.T) {
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
			_, err := c.ListGroups(t.Context(), ListGroupsParams{})
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
			if !strings.HasPrefix(err.Error(), "list groups: ") {
				t.Errorf("err = %q, want list groups prefix", err)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}
