package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type fakePolicyAPI struct {
	policies []netbird.Policy
	err      error
	calls    int
}

func (f *fakePolicyAPI) ListPolicies(context.Context) ([]netbird.Policy, error) {
	f.calls++
	return f.policies, f.err
}

func TestToListPoliciesOutput(t *testing.T) {
	tests := []struct {
		name string
		in   []netbird.Policy
		want string
	}{
		{name: "nil", in: nil, want: `{"policies":[]}`},
		{name: "empty", in: []netbird.Policy{}, want: `{"policies":[]}`},
		{
			name: "nil lists and optional fields absent",
			in:   []netbird.Policy{{ID: "p1", Name: "n", Enabled: true}},
			want: `{"policies":[{"id":"p1","name":"n","enabled":true,"source_posture_checks":[],"rules":[]}]}`,
		},
		{
			name: "rule with nil lists",
			in:   []netbird.Policy{{ID: "p1", Name: "n", Rules: []netbird.PolicyRule{{Name: "r", Action: "accept", Protocol: "all"}}}},
			want: `{"policies":[{"id":"p1","name":"n","enabled":false,"source_posture_checks":[],` +
				`"rules":[{"name":"r","enabled":false,"action":"accept","bidirectional":false,"protocol":"all","sources":[],"destinations":[]}]}]}`,
		},
		{
			name: "empty ports and nil authorized groups omitted",
			in: []netbird.Policy{{ID: "p1", Rules: []netbird.PolicyRule{{
				Ports: []string{}, PortRanges: []netbird.RulePortRange{}, AuthorizedGroups: nil,
				Sources: []netbird.Group{}, Destinations: []netbird.Group{},
			}}}},
			want: `{"policies":[{"id":"p1","name":"","enabled":false,"source_posture_checks":[],` +
				`"rules":[{"name":"","enabled":false,"action":"","bidirectional":false,"protocol":"","sources":[],"destinations":[]}]}]}`,
		},
		{
			name: "full mapping and order kept",
			in: []netbird.Policy{
				{
					ID: "p2", Name: "b", Description: "d", Enabled: true, SourcePostureChecks: []string{"pc2", "pc1"},
					Rules: []netbird.PolicyRule{
						{
							ID: "r2", Name: "two", Description: "rd", Enabled: true, Action: "accept", Bidirectional: true,
							Protocol: "tcp", Ports: []string{"80", "443"},
							PortRanges:          []netbird.RulePortRange{{Start: 8000, End: 8100}},
							AuthorizedGroups:    map[string][]string{"g9": {"root", "dev"}},
							Sources:             []netbird.Group{{ID: "g1", Name: "src", PeersCount: 5, ResourcesCount: 2, Issued: "api"}},
							SourceResource:      &netbird.Resource{ID: "r1", Type: "host"},
							Destinations:        []netbird.Group{{ID: "g2", Name: "dst"}, {ID: "g3", Name: "dst2"}},
							DestinationResource: &netbird.Resource{ID: "r3", Type: "subnet"},
						},
						{ID: "r1", Name: "one", Action: "drop", Protocol: "udp"},
					},
				},
				{ID: "p1", Name: "a"},
			},
			want: `{"policies":[{"id":"p2","name":"b","description":"d","enabled":true,"source_posture_checks":["pc2","pc1"],"rules":[` +
				`{"id":"r2","name":"two","description":"rd","enabled":true,"action":"accept","bidirectional":true,"protocol":"tcp",` +
				`"ports":["80","443"],"port_ranges":[{"start":8000,"end":8100}],` +
				`"sources":[{"id":"g1","name":"src"}],"source_resource":{"id":"r1","type":"host"},` +
				`"destinations":[{"id":"g2","name":"dst"},{"id":"g3","name":"dst2"}],"destination_resource":{"id":"r3","type":"subnet"},` +
				`"authorized_groups":{"g9":["root","dev"]}},` +
				`{"id":"r1","name":"one","enabled":false,"action":"drop","bidirectional":false,"protocol":"udp","sources":[],"destinations":[]}]},` +
				`{"id":"p1","name":"a","enabled":false,"source_posture_checks":[],"rules":[]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(toListPoliciesOutput(tt.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestListPoliciesTool(t *testing.T) {
	full := []netbird.Policy{
		{ID: "a", Rules: []netbird.PolicyRule{{ID: "r", SourceResource: &netbird.Resource{ID: "x", Type: "host"}}}},
		{ID: "b"},
	}
	tests := []struct {
		name         string
		api          *fakePolicyAPI
		wantErr      string
		wantPolicies int
	}{
		{name: "success", api: &fakePolicyAPI{policies: full}, wantPolicies: 2},
		{name: "empty", api: &fakePolicyAPI{}},
		{
			name:    "401",
			api:     &fakePolicyAPI{err: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{policyAPI: tt.api})
			res, text := callTool(t, cs, "list_policies", nil)

			if tt.api.calls != 1 {
				t.Errorf("api calls = %d, want 1", tt.api.calls)
			}
			if tt.wantErr != "" {
				if !res.IsError || text != tt.wantErr {
					t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
				}
				if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "is_error=true") {
					t.Errorf("want Warn log with is_error=true, got %s", logs)
				}
			} else {
				if res.IsError {
					t.Fatalf("unexpected tool error: %s", text)
				}
				sc, ok := res.StructuredContent.(map[string]any)
				if !ok {
					t.Fatalf("structured content = %#v", res.StructuredContent)
				}
				policies, ok := sc["policies"].([]any)
				if !ok || len(policies) != tt.wantPolicies {
					t.Fatalf("policies = %#v, want %d non-nil", sc["policies"], tt.wantPolicies)
				}
				if tt.wantPolicies == 2 {
					rules, _ := policies[0].(map[string]any)["rules"].([]any)
					if len(rules) != 1 {
						t.Fatalf("rules = %#v", rules)
					}
					rule := rules[0].(map[string]any)
					res, _ := rule["source_resource"].(map[string]any)
					if res["id"] != "x" || res["type"] != "host" {
						t.Errorf("source_resource = %#v", rule["source_resource"])
					}
					for _, k := range []string{"sources", "destinations"} {
						if l, ok := rule[k].([]any); !ok || len(l) != 0 {
							t.Errorf("%s = %#v, want empty non-nil", k, rule[k])
						}
					}
					for _, k := range []string{"destination_resource", "authorized_groups", "ports", "port_ranges", "description"} {
						if _, has := rule[k]; has {
							t.Errorf("%q present when absent", k)
						}
					}
					second := policies[1].(map[string]any)
					for _, k := range []string{"rules", "source_posture_checks"} {
						if l, ok := second[k].([]any); !ok || len(l) != 0 {
							t.Errorf("%s = %#v, want empty non-nil", k, second[k])
						}
					}
				}
				if !strings.Contains(logs.String(), "tool=list_policies") || !strings.Contains(logs.String(), "is_error=false") {
					t.Errorf("logs missing tool fields: %s", logs)
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListPoliciesToolLeaksNoToken(t *testing.T) {
	t.Parallel()
	api := &fakePolicyAPI{
		policies: []netbird.Policy{{ID: "a", Name: "n"}},
	}
	cs, logs := connectFake(t, fakeClient{policyAPI: api})
	res, text := callTool(t, cs, "list_policies", nil)
	raw, _ := json.Marshal(res.StructuredContent)
	if res.IsError || strings.Contains(text, sentinelToken) || strings.Contains(string(raw), sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("IsError=%v text=%q logs=%s", res.IsError, text, logs)
	}
}

func TestListPoliciesToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := listPolicies(&fakePolicyAPI{})
	if tool.Name != "list_policies" {
		t.Errorf("name = %q", tool.Name)
	}
	if n := reflect.TypeFor[listPoliciesInput]().NumField(); n != 0 {
		t.Errorf("input has %d fields, want 0", n)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
