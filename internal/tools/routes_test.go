package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type fakeRouteAPI struct {
	routes []netbird.Route
	err    error
	calls  int
}

func (f *fakeRouteAPI) ListRoutes(context.Context) ([]netbird.Route, error) {
	f.calls++
	return f.routes, f.err
}

func TestToListRoutesOutput(t *testing.T) {
	t.Parallel()
	groupNames := map[string]string{"g1": "devs", "g2": "ops", "g3": "web", "g4": "admins"}
	peerNames := map[string]string{"p1": "laptop"}
	tests := []struct {
		name string
		in   []netbird.Route
		want string
	}{
		{name: "nil", in: nil, want: `{"routes":[]}`},
		{name: "empty", in: []netbird.Route{}, want: `{"routes":[]}`},
		{
			name: "nil lists and optional fields absent",
			in:   []netbird.Route{{ID: "r1", NetworkID: "n", NetworkType: "IPv4"}},
			want: `{"routes":[{"id":"r1","network_id":"n","enabled":false,"network_type":"IPv4","domains":[],"peer_groups":[],"groups":[],"access_control_groups":[],"metric":0,"masquerade":false,"keep_route":false,"skip_auto_apply":false}]}`,
		},
		{
			name: "all fields, names resolved, order kept",
			in: []netbird.Route{
				{
					ID: "r2", NetworkType: "Domain", Description: "d", NetworkID: "net2", Enabled: true,
					Peer: "p1", PeerGroups: []string{"g2", "g1"}, Network: "10.0.0.0/24", Domains: []string{"a.io", "b.io"},
					Metric: 200, Masquerade: true, Groups: []string{"g3"}, KeepRoute: true,
					AccessControlGroups: []string{"g4", "g1"}, SkipAutoApply: true,
				},
				{ID: "r1", Domains: []string{}, PeerGroups: []string{"g1"}},
			},
			want: `{"routes":[{"id":"r2","network_id":"net2","description":"d","enabled":true,"network_type":"Domain","network":"10.0.0.0/24",` +
				`"domains":["a.io","b.io"],"peer":{"id":"p1","name":"laptop"},"peer_groups":[{"id":"g2","name":"ops"},{"id":"g1","name":"devs"}],` +
				`"groups":[{"id":"g3","name":"web"}],"access_control_groups":[{"id":"g4","name":"admins"},{"id":"g1","name":"devs"}],` +
				`"metric":200,"masquerade":true,"keep_route":true,"skip_auto_apply":true},` +
				`{"id":"r1","network_id":"","enabled":false,"network_type":"","domains":[],"peer_groups":[{"id":"g1","name":"devs"}],"groups":[],"access_control_groups":[],"metric":0,"masquerade":false,"keep_route":false,"skip_auto_apply":false}]}`,
		},
		{
			name: "unknown ids have no name",
			in:   []netbird.Route{{ID: "r1", Peer: "gone", PeerGroups: []string{"gone", "g1"}, Groups: []string{"gone"}, AccessControlGroups: []string{"gone"}}},
			want: `{"routes":[{"id":"r1","network_id":"","enabled":false,"network_type":"","domains":[],"peer":{"id":"gone"},` +
				`"peer_groups":[{"id":"gone"},{"id":"g1","name":"devs"}],"groups":[{"id":"gone"}],"access_control_groups":[{"id":"gone"}],` +
				`"metric":0,"masquerade":false,"keep_route":false,"skip_auto_apply":false}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(toListRoutesOutput(tt.in, groupNames, peerNames))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestListRoutesTool(t *testing.T) {
	tests := []struct {
		name       string
		api        *fakeRouteAPI
		wantErr    string
		wantRoutes int
	}{
		{
			name: "success",
			api: &fakeRouteAPI{routes: []netbird.Route{
				{ID: "a", Description: "d", Network: "10.0.0.0/8", Peer: "p", Metric: 9999},
				{ID: "b"},
			}},
			wantRoutes: 2,
		},
		{name: "empty", api: &fakeRouteAPI{}},
		{
			name:    "401",
			api:     &fakeRouteAPI{err: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{routeAPI: tt.api, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
			res, text := callTool(t, cs, "list_routes", nil)

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
				routes, ok := sc["routes"].([]any)
				if !ok || len(routes) != tt.wantRoutes {
					t.Fatalf("routes = %#v, want %d non-nil", sc["routes"], tt.wantRoutes)
				}
				if tt.wantRoutes == 2 {
					first := routes[0].(map[string]any)
					second := routes[1].(map[string]any)
					if first["description"] != "d" || first["network"] != "10.0.0.0/8" || first["metric"] != float64(9999) {
						t.Errorf("first route = %#v", first)
					}
					if peer, ok := first["peer"].(map[string]any); !ok || peer["id"] != "p" {
						t.Errorf("first peer = %#v", first["peer"])
					}
					for _, k := range []string{"description", "network", "peer"} {
						if _, has := second[k]; has {
							t.Errorf("%q present when empty: %#v", k, second)
						}
					}
					for _, k := range []string{"domains", "peer_groups", "groups", "access_control_groups"} {
						if l, ok := second[k].([]any); !ok || len(l) != 0 {
							t.Errorf("%s = %#v, want empty non-nil", k, second[k])
						}
					}
				}
				if !strings.Contains(logs.String(), "tool=list_routes") || !strings.Contains(logs.String(), "is_error=false") {
					t.Errorf("logs missing tool fields: %s", logs)
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListRoutesToolResolvesNames(t *testing.T) {
	t.Parallel()
	api := &fakeRouteAPI{routes: []netbird.Route{{
		ID: "r1", NetworkID: "n", NetworkType: "IPv4", Peer: "p1",
		PeerGroups: []string{"g1"}, Groups: []string{"g2"}, AccessControlGroups: []string{"g1", "gone"},
	}}}
	grp := &fakeGroupAPI{groups: []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "ops"}}}
	peers := &fakePeerAPI{peers: []netbird.Peer{{ID: "p1", Name: "laptop"}}}
	cs, logs := connectFake(t, fakeClient{routeAPI: api, groupAPI: grp, peerAPI: peers})
	res, text := callTool(t, cs, "list_routes", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	raw := structuredJSON(t, res)
	assertJSONEqual(t, raw, `{"routes":[{"id":"r1","network_id":"n","enabled":false,"network_type":"IPv4","domains":[],`+
		`"peer":{"id":"p1","name":"laptop"},"peer_groups":[{"id":"g1","name":"devs"}],"groups":[{"id":"g2","name":"ops"}],`+
		`"access_control_groups":[{"id":"g1","name":"devs"},{"id":"gone"}],"metric":0,"masquerade":false,"keep_route":false,"skip_auto_apply":false}]}`)
	if api.calls != 1 || grp.calls != 1 || peers.listCalls != 1 {
		t.Errorf("calls = %d/%d/%d, want 1/1/1", api.calls, grp.calls, peers.listCalls)
	}
	if peers.params != (netbird.ListPeersParams{}) || grp.params != (netbird.ListGroupsParams{}) {
		t.Errorf("lookup params = %+v / %+v, want empty", peers.params, grp.params)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(raw, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestListRoutesToolDegrades(t *testing.T) {
	t.Parallel()
	routes := []netbird.Route{{ID: "r1", NetworkID: "n", NetworkType: "IPv4", Peer: "p1", PeerGroups: []string{"g1"}}}
	groups := []netbird.Group{{ID: "g1", Name: "devs"}}
	peers := []netbird.Peer{{ID: "p1", Name: "laptop"}}
	boom := &netbird.APIError{Status: 500, Message: "boom"}
	const boomMsg = "netbird api: 500: boom (hint: NetBird server error, retry later)"
	const tail = `"domains":[],"metric":0,"masquerade":false,"keep_route":false,"skip_auto_apply":false`
	tests := []struct {
		name string
		grp  *fakeGroupAPI
		peer *fakePeerAPI
		want string
	}{
		{
			name: "groups failure",
			grp:  &fakeGroupAPI{err: boom},
			peer: &fakePeerAPI{peers: peers},
			want: `{"routes":[{"id":"r1","network_id":"n","enabled":false,"network_type":"IPv4",` + tail +
				`,"peer":{"id":"p1","name":"laptop"},"peer_groups":[{"id":"g1"}],"groups":[],"access_control_groups":[]}],"names_error":"` + boomMsg + `"}`,
		},
		{
			name: "peers failure",
			grp:  &fakeGroupAPI{groups: groups},
			peer: &fakePeerAPI{listErr: boom},
			want: `{"routes":[{"id":"r1","network_id":"n","enabled":false,"network_type":"IPv4",` + tail +
				`,"peer":{"id":"p1"},"peer_groups":[{"id":"g1","name":"devs"}],"groups":[],"access_control_groups":[]}],"names_error":"` + boomMsg + `"}`,
		},
		{
			name: "both fail",
			grp:  &fakeGroupAPI{err: boom},
			peer: &fakePeerAPI{listErr: boom},
			want: `{"routes":[{"id":"r1","network_id":"n","enabled":false,"network_type":"IPv4",` + tail +
				`,"peer":{"id":"p1"},"peer_groups":[{"id":"g1"}],"groups":[],"access_control_groups":[]}],"names_error":"` + boomMsg + `"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeRouteAPI{routes: routes}
			cs, logs := connectFake(t, fakeClient{routeAPI: api, groupAPI: tt.grp, peerAPI: tt.peer})
			res, text := callTool(t, cs, "list_routes", nil)
			if res.IsError {
				t.Fatalf("lookup failure must not fail the tool: %s", text)
			}
			raw := structuredJSON(t, res)
			assertJSONEqual(t, raw, tt.want)
			if strings.Contains(text, sentinelToken) || strings.Contains(raw, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListRoutesToolCoreFailureFails(t *testing.T) {
	t.Parallel()
	api := &fakeRouteAPI{err: &netbird.APIError{Status: 401, Message: "bad token"}}
	grp := &fakeGroupAPI{groups: []netbird.Group{{ID: "g1", Name: "devs"}}}
	peers := &fakePeerAPI{peers: []netbird.Peer{{ID: "p1", Name: "laptop"}}}
	cs, logs := connectFake(t, fakeClient{routeAPI: api, groupAPI: grp, peerAPI: peers})
	res, text := callTool(t, cs, "list_routes", nil)
	const want = "netbird api: 401: bad token (hint: token invalid or expired)"
	if !res.IsError || text != want {
		t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, want)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestListRoutesToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := listRoutes(fakeClient{routeAPI: &fakeRouteAPI{}})
	if tool.Name != "list_routes" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Lists all network routes in the NetBird account: network or domains, routing peer or peer groups, access control groups, metric and masquerade. Group and peer references include names; names_error appears when names cannot be fetched."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	if n := reflect.TypeFor[listRoutesInput]().NumField(); n != 0 {
		t.Errorf("input has %d fields, want 0", n)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
