package netbird

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListPoliciesRequest(t *testing.T) {
	t.Parallel()
	var got *http.Request
	c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte("[]"))
	})
	if _, err := c.ListPolicies(t.Context()); err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.Method)
	}
	if got.URL.Path != "/api/policies" {
		t.Errorf("path = %q, want /api/policies", got.URL.Path)
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

func TestListPoliciesDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Policy
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []Policy{}},
		{
			name: "full policy with unknown fields",
			body: `[{"id":"pol1","name":"Default","description":"allow all","enabled":true,
				"source_posture_checks":["pc1","pc2"],"future_field":{"x":1},
				"rules":[{"id":"rule1","name":"web","description":"web traffic","enabled":true,
					"action":"accept","bidirectional":false,"protocol":"tcp",
					"ports":["80","443"],
					"port_ranges":[{"start":8000,"end":8080,"future_range_field":1}],
					"authorized_groups":{"g9":["root","admin"]},
					"sources":[{"id":"g1","name":"devs","peers_count":2,"resources_count":0,"issued":"api","future_group_field":1}],
					"sourceResource":{"id":"r1","type":"host"},
					"destinations":[{"id":"g2","name":"servers","peers_count":1,"resources_count":3,"issued":"jwt"}],
					"destinationResource":{"id":"r2","type":"subnet"}}]}]`,
			want: []Policy{{
				ID: "pol1", Name: "Default", Description: "allow all", Enabled: true,
				SourcePostureChecks: []string{"pc1", "pc2"},
				Rules: []PolicyRule{{
					ID: "rule1", Name: "web", Description: "web traffic", Enabled: true,
					Action: "accept", Bidirectional: false, Protocol: "tcp",
					Ports:               []string{"80", "443"},
					PortRanges:          []RulePortRange{{Start: 8000, End: 8080}},
					AuthorizedGroups:    map[string][]string{"g9": {"root", "admin"}},
					Sources:             []Group{{ID: "g1", Name: "devs", PeersCount: 2, Issued: "api"}},
					SourceResource:      &Resource{ID: "r1", Type: "host"},
					Destinations:        []Group{{ID: "g2", Name: "servers", PeersCount: 1, ResourcesCount: 3, Issued: "jwt"}},
					DestinationResource: &Resource{ID: "r2", Type: "subnet"},
				}},
			}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"pol2","name":"min","enabled":false,"source_posture_checks":[],
				"rules":[{"name":"r","enabled":true,"action":"drop","bidirectional":true,"protocol":"all"}]}]`,
			want: []Policy{{
				ID: "pol2", Name: "min", SourcePostureChecks: []string{},
				Rules: []PolicyRule{{Name: "r", Enabled: true, Action: "drop", Bidirectional: true, Protocol: "all"}},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.ListPolicies(t.Context())
			if err != nil {
				t.Fatalf("ListPolicies: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListPoliciesAPIError(t *testing.T) {
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
			_, err := c.ListPolicies(t.Context())
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
			if !strings.HasPrefix(err.Error(), "list policies: ") {
				t.Errorf("err = %q, want list policies prefix", err)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}
