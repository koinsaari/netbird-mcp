package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type fakeGroupAPI struct {
	groups []netbird.Group
	err    error
	calls  int
	params netbird.ListGroupsParams
}

func (f *fakeGroupAPI) ListGroups(_ context.Context, p netbird.ListGroupsParams) ([]netbird.Group, error) {
	f.calls++
	f.params = p
	return f.groups, f.err
}

func TestToListGroupsOutput(t *testing.T) {
	tests := []struct {
		name string
		in   []netbird.Group
		want string
	}{
		{name: "nil", in: nil, want: `{"groups":[]}`},
		{name: "empty", in: []netbird.Group{}, want: `{"groups":[]}`},
		{
			name: "issued absent",
			in:   []netbird.Group{{ID: "g1", Name: "All", PeersCount: 3, ResourcesCount: 1}},
			want: `{"groups":[{"id":"g1","name":"All","peers_count":3,"resources_count":1}]}`,
		},
		{
			name: "members dropped and order kept",
			in: []netbird.Group{
				{ID: "g2", Name: "b", Issued: "api", Peers: []netbird.PeerMinimum{{ID: "p", Name: "n"}}, Resources: []netbird.Resource{{ID: "r", Type: "host"}}},
				{ID: "g1", Name: "a", Issued: "jwt"},
			},
			want: `{"groups":[{"id":"g2","name":"b","peers_count":0,"resources_count":0,"issued":"api"},{"id":"g1","name":"a","peers_count":0,"resources_count":0,"issued":"jwt"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(toListGroupsOutput(tt.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestListGroupsTool(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		api        *fakeGroupAPI
		wantErr    string
		wantParams netbird.ListGroupsParams
		wantGroups int
	}{
		{
			name:       "success",
			api:        &fakeGroupAPI{groups: []netbird.Group{{ID: "a"}, {ID: "b", Issued: "api"}}},
			wantGroups: 2,
		},
		{name: "empty", api: &fakeGroupAPI{}},
		{
			name:       "name passed through",
			args:       map[string]any{"name": "distinctname"},
			api:        &fakeGroupAPI{},
			wantParams: netbird.ListGroupsParams{Name: "distinctname"},
		},
		{
			name:    "401",
			api:     &fakeGroupAPI{err: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectClient(t, nil, tt.api)
			res, text := callTool(t, cs, "list_groups", tt.args)

			if tt.api.calls != 1 {
				t.Errorf("api calls = %d, want 1", tt.api.calls)
			}
			if tt.api.params != tt.wantParams {
				t.Errorf("params = %+v, want %+v", tt.api.params, tt.wantParams)
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
				groups, ok := sc["groups"].([]any)
				if !ok || len(groups) != tt.wantGroups {
					t.Errorf("groups = %#v, want %d non-nil", sc["groups"], tt.wantGroups)
				}
				if tt.wantGroups == 2 {
					first := groups[0].(map[string]any)
					if _, has := first["issued"]; has {
						t.Errorf("issued present when empty: %#v", first)
					}
					for _, k := range []string{"peers", "resources"} {
						if _, has := first[k]; has {
							t.Errorf("member list %q in output", k)
						}
					}
				}
				if !strings.Contains(logs.String(), "tool=list_groups") || !strings.Contains(logs.String(), "is_error=false") {
					t.Errorf("logs missing tool fields: %s", logs)
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
			if strings.Contains(logs.String(), "distinctname") {
				t.Errorf("args leaked into logs: %s", logs)
			}
		})
	}
}

func TestListGroupsToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := listGroups(&fakeGroupAPI{})
	if tool.Name != "list_groups" {
		t.Errorf("name = %q", tool.Name)
	}
	want := reflect.TypeFor[listGroupsInput]()
	if f, _ := want.FieldByName("Name"); f.Tag.Get("json") != "name,omitempty" {
		t.Errorf("name tag = %q", f.Tag)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
