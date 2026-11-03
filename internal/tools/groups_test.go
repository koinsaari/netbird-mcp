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

	group    netbird.Group
	getErr   error
	getCalls int
	id       string
}

func (f *fakeGroupAPI) ListGroups(_ context.Context, p netbird.ListGroupsParams) ([]netbird.Group, error) {
	f.calls++
	f.params = p
	return f.groups, f.err
}

func (f *fakeGroupAPI) GetGroup(_ context.Context, id string) (netbird.Group, error) {
	f.getCalls++
	f.id = id
	return f.group, f.getErr
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

func TestToGroupDetailsOutput(t *testing.T) {
	tests := []struct {
		name string
		in   netbird.Group
		want string
	}{
		{
			name: "nil lists and issued absent",
			in:   netbird.Group{ID: "g1", Name: "All"},
			want: `{"id":"g1","name":"All","peers_count":0,"resources_count":0,"peers":[],"resources":[]}`,
		},
		{
			name: "empty lists",
			in:   netbird.Group{ID: "g1", Name: "All", Peers: []netbird.PeerMinimum{}, Resources: []netbird.Resource{}},
			want: `{"id":"g1","name":"All","peers_count":0,"resources_count":0,"peers":[],"resources":[]}`,
		},
		{
			name: "members mapped and order kept",
			in: netbird.Group{
				ID: "g2", Name: "b", PeersCount: 2, ResourcesCount: 2, Issued: "api",
				Peers:     []netbird.PeerMinimum{{ID: "p2", Name: "two"}, {ID: "p1", Name: "one"}},
				Resources: []netbird.Resource{{ID: "r2", Type: "host"}, {ID: "r1", Type: "subnet"}},
			},
			want: `{"id":"g2","name":"b","peers_count":2,"resources_count":2,"issued":"api",` +
				`"peers":[{"id":"p2","name":"two"},{"id":"p1","name":"one"}],` +
				`"resources":[{"id":"r2","type":"host"},{"id":"r1","type":"subnet"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(toGroupDetailsOutput(tt.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetGroupTool(t *testing.T) {
	const id = "cdl0ajd5jtbs73abcdef"
	full := netbird.Group{
		ID: id, Name: "devs", PeersCount: 1, ResourcesCount: 1, Issued: "api",
		Peers:     []netbird.PeerMinimum{{ID: "p1", Name: "laptop"}},
		Resources: []netbird.Resource{{ID: "r1", Type: "host"}},
	}
	notFound := &netbird.APIError{Status: 404, Message: "group not found"}
	invalid := errInvalidGroupID.Error()
	tests := []struct {
		name      string
		args      map[string]any
		api       *fakeGroupAPI
		wantErr   string
		wantCalls int
	}{
		{name: "success", args: map[string]any{"group_id": id}, api: &fakeGroupAPI{group: full}, wantCalls: 1},
		{name: "empty id", args: map[string]any{"group_id": ""}, api: &fakeGroupAPI{}, wantErr: invalid},
		{name: "19 chars", args: map[string]any{"group_id": id[:19]}, api: &fakeGroupAPI{}, wantErr: invalid},
		{name: "21 chars", args: map[string]any{"group_id": id + "a"}, api: &fakeGroupAPI{}, wantErr: invalid},
		{name: "uppercase", args: map[string]any{"group_id": "CDL0AJD5JTBS73ABCDEF"}, api: &fakeGroupAPI{}, wantErr: invalid},
		{name: "contains w", args: map[string]any{"group_id": "cdl0ajd5jtbs73abcdew"}, api: &fakeGroupAPI{}, wantErr: invalid},
		{name: "path traversal", args: map[string]any{"group_id": "../x"}, api: &fakeGroupAPI{}, wantErr: invalid},
		{
			name: "404", args: map[string]any{"group_id": id}, api: &fakeGroupAPI{getErr: notFound}, wantCalls: 1,
			wantErr: "netbird api: 404: group not found (hint: not found; check the ID or NETBIRD_API_URL)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectClient(t, nil, tt.api)
			res, text := callTool(t, cs, "get_group", tt.args)

			if tt.api.getCalls != tt.wantCalls {
				t.Errorf("api calls = %d, want %d", tt.api.getCalls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && tt.api.id != id {
				t.Errorf("api id = %q, want %q", tt.api.id, id)
			}
			if tt.wantErr != "" {
				if !res.IsError || text != tt.wantErr {
					t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
				}
			} else {
				if res.IsError {
					t.Fatalf("unexpected tool error: %s", text)
				}
				sc, ok := res.StructuredContent.(map[string]any)
				if !ok {
					t.Fatalf("structured content = %#v", res.StructuredContent)
				}
				for _, k := range []string{"id", "name", "peers_count", "resources_count", "issued", "peers", "resources"} {
					if _, ok := sc[k]; !ok {
						t.Errorf("structured content missing %q", k)
					}
				}
				peers, _ := sc["peers"].([]any)
				if len(peers) != 1 || peers[0].(map[string]any)["name"] != "laptop" {
					t.Errorf("peers = %#v", sc["peers"])
				}
				resources, _ := sc["resources"].([]any)
				if len(resources) != 1 || resources[0].(map[string]any)["type"] != "host" {
					t.Errorf("resources = %#v", sc["resources"])
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestGetGroupToolLeaksNoToken(t *testing.T) {
	t.Parallel()
	g := &fakeGroupAPI{getErr: &netbird.APIError{Status: 500, Message: "boom"}}
	cs, logs := connectClient(t, nil, g)
	res, text := callTool(t, cs, "get_group", map[string]any{"group_id": "cdl0ajd5jtbs73abcdef"})
	if !res.IsError || strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("IsError=%v text=%q logs=%s", res.IsError, text, logs)
	}
}

func TestGetGroupToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := getGroup(&fakeGroupAPI{})
	if tool.Name != "get_group" {
		t.Errorf("name = %q", tool.Name)
	}
	f, _ := reflect.TypeFor[getGroupInput]().FieldByName("GroupID")
	if f.Tag.Get("json") != "group_id" {
		t.Errorf("group_id tag = %q", f.Tag)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
