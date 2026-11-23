package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestToListNetworksOutput(t *testing.T) {
	t.Parallel()
	networks := []netbird.Network{
		{ID: "n2", Name: "second", Description: "d", RoutingPeersCount: 2, Policies: []string{"p1"}, Routers: []string{"r1"}, Resources: []string{"x1"}},
		{ID: "n1", Name: "first"},
	}
	routers := [][]netbird.NetworkRouter{
		{
			{ID: "r1", Peer: "peer1", Metric: 100, Masquerade: true, Enabled: true},
			{ID: "r2", PeerGroups: []string{"g1", "gone"}, Metric: 200},
			{ID: "r3", Peer: "gone", Metric: 300},
		},
		nil,
	}
	resources := [][]netbird.NetworkResource{
		{
			{ID: "x1", Name: "web", Description: "rd", Address: "10.0.0.0/24", Type: "subnet", Enabled: true,
				Groups: []netbird.Group{{ID: "g1", Name: "devs", PeersCount: 5, Issued: "api"}}},
			{ID: "x2", Name: "db", Address: "db.example.com", Type: "domain"},
		},
		nil,
	}

	groupNames := map[string]string{"g1": "devs"}
	peerNames := map[string]string{"peer1": "laptop"}
	got := toListNetworksOutput(networks, routers, resources, refNames{groups: groupNames, peers: peerNames})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const want = `{"networks":[` +
		`{"id":"n2","name":"second","description":"d","routing_peers_count":2,"policies":["p1"],` +
		`"routers":[{"id":"r1","peer":{"id":"peer1","name":"laptop"},"peer_groups":[],"metric":100,"masquerade":true,"enabled":true},` +
		`{"id":"r2","peer_groups":[{"id":"g1","name":"devs"},{"id":"gone"}],"metric":200,"masquerade":false,"enabled":false},` +
		`{"id":"r3","peer":{"id":"gone"},"peer_groups":[],"metric":300,"masquerade":false,"enabled":false}],` +
		`"resources":[{"id":"x1","name":"web","description":"rd","address":"10.0.0.0/24","type":"subnet","enabled":true,"groups":[{"id":"g1","name":"devs"}]},` +
		`{"id":"x2","name":"db","address":"db.example.com","type":"domain","enabled":false,"groups":[]}]},` +
		`{"id":"n1","name":"first","routing_peers_count":0,"policies":[],"routers":[],"resources":[]}]}`
	if string(b) != want {
		t.Errorf("output =\n%s\nwant\n%s", b, want)
	}
}

func TestToListNetworksOutputEmpty(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(toListNetworksOutput(nil, nil, nil, refNames{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `{"networks":[]}` {
		t.Errorf("output = %s", b)
	}
}

type fakeNetworkAPI struct {
	networks     []netbird.Network
	routers      map[string][]netbird.NetworkRouter
	resources    map[string][]netbird.NetworkResource
	listErr      error
	routersErr   error
	resourcesErr error

	listCalls      atomic.Int32
	routersCalls   atomic.Int32
	resourcesCalls atomic.Int32

	mu          sync.Mutex
	routerIDs   map[string]int
	resourceIDs map[string]int
}

func (f *fakeNetworkAPI) count(m *map[string]int, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if *m == nil {
		*m = map[string]int{}
	}
	(*m)[id]++
}

func (f *fakeNetworkAPI) ListNetworks(context.Context) ([]netbird.Network, error) {
	f.listCalls.Add(1)
	return f.networks, f.listErr
}

func (f *fakeNetworkAPI) ListNetworkRouters(_ context.Context, id string) ([]netbird.NetworkRouter, error) {
	f.routersCalls.Add(1)
	f.count(&f.routerIDs, id)
	if f.routersErr != nil {
		return nil, f.routersErr
	}
	return f.routers[id], nil
}

func (f *fakeNetworkAPI) ListNetworkResources(_ context.Context, id string) ([]netbird.NetworkResource, error) {
	f.resourcesCalls.Add(1)
	f.count(&f.resourceIDs, id)
	if f.resourcesErr != nil {
		return nil, f.resourcesErr
	}
	return f.resources[id], nil
}

func TestListNetworksTool(t *testing.T) {
	t.Parallel()
	f := &fakeNetworkAPI{
		networks: []netbird.Network{
			{ID: "n1", Name: "a", Routers: []string{"r1"}, Resources: []string{"x1"}},
			{ID: "n2", Name: "b", RoutingPeersCount: 1, Routers: []string{"r2"}},
		},
		routers: map[string][]netbird.NetworkRouter{
			"n1": {{ID: "r1", Peer: "p1", Enabled: true}},
			"n2": {{ID: "r2", PeerGroups: []string{"g1"}}},
		},
		resources: map[string][]netbird.NetworkResource{
			"n1": {{ID: "x1", Name: "web", Address: "10.0.0.0/24", Type: "subnet", Groups: []netbird.Group{{ID: "g1", Name: "devs"}}}},
		},
	}
	grp := &fakeGroupAPI{groups: []netbird.Group{{ID: "g1", Name: "devs"}}}
	peers := &fakePeerAPI{peers: []netbird.Peer{{ID: "p1", Name: "laptop"}}}
	cs, logs := connectFake(t, fakeClient{networkAPI: f, groupAPI: grp, peerAPI: peers})
	res, text := callTool(t, cs, "list_networks", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	raw := structuredJSON(t, res)
	var out listNetworksOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Networks) != 2 || out.Networks[0].ID != "n1" || out.Networks[1].ID != "n2" {
		t.Fatalf("networks = %+v", out.Networks)
	}
	if got := out.Networks[0].Routers; len(got) != 1 || got[0].ID != "r1" || !reflect.DeepEqual(got[0].Peer, &refOutput{ID: "p1", Name: "laptop"}) {
		t.Errorf("n1 routers = %+v", got)
	}
	if got := out.Networks[1].Routers; len(got) != 1 || got[0].ID != "r2" || !reflect.DeepEqual(got[0].PeerGroups, []refOutput{{ID: "g1", Name: "devs"}}) {
		t.Errorf("n2 routers = %+v", got)
	}
	if got := out.Networks[0].Resources; len(got) != 1 || got[0].ID != "x1" || len(got[0].Groups) != 1 {
		t.Errorf("n1 resources = %+v", got)
	}
	if len(out.Networks[1].Resources) != 0 {
		t.Errorf("n2 resources = %+v", out.Networks[1].Resources)
	}
	if f.listCalls.Load() != 1 || f.routersCalls.Load() != 2 || f.resourcesCalls.Load() != 1 {
		t.Errorf("calls = %d/%d/%d, want 1/2/1", f.listCalls.Load(), f.routersCalls.Load(), f.resourcesCalls.Load())
	}
	if !strings.Contains(logs.String(), "tool=list_networks") || !strings.Contains(logs.String(), "is_error=false") {
		t.Errorf("logs missing tool fields: %s", logs)
	}
	if grp.calls != 1 || peers.listCalls != 1 {
		t.Errorf("lookup calls = %d/%d, want 1/1", grp.calls, peers.listCalls)
	}
	if grp.params != (netbird.ListGroupsParams{}) || peers.params != (netbird.ListPeersParams{}) {
		t.Errorf("lookup params = %+v / %+v, want empty", grp.params, peers.params)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(raw, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestListNetworksToolEmpty(t *testing.T) {
	t.Parallel()
	f := &fakeNetworkAPI{}
	cs, _ := connectFake(t, fakeClient{networkAPI: f, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	res, text := callTool(t, cs, "list_networks", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	sc, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content = %#v", res.StructuredContent)
	}
	nets, ok := sc["networks"].([]any)
	if !ok || len(nets) != 0 {
		t.Errorf("networks = %#v, want empty non-nil", sc["networks"])
	}
	if f.routersCalls.Load() != 0 || f.resourcesCalls.Load() != 0 {
		t.Errorf("sub-calls = %d/%d, want 0/0", f.routersCalls.Load(), f.resourcesCalls.Load())
	}
}

func TestListNetworksToolErrors(t *testing.T) {
	t.Parallel()
	nets := []netbird.Network{{ID: "n1", Routers: []string{"r"}, Resources: []string{"x"}}, {ID: "n2", Routers: []string{"r"}, Resources: []string{"x"}}}
	tests := []struct {
		name    string
		api     *fakeNetworkAPI
		wantErr string
	}{
		{
			name:    "list fails with 401",
			api:     &fakeNetworkAPI{listErr: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
		{
			name:    "routers fail",
			api:     &fakeNetworkAPI{networks: nets, routersErr: &netbird.APIError{Status: 503, Message: "down"}},
			wantErr: "netbird api: 503: down (hint: NetBird server error, retry later)",
		},
		{
			name:    "resources fail",
			api:     &fakeNetworkAPI{networks: nets, resourcesErr: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{networkAPI: tt.api, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
			res, text := callTool(t, cs, "list_networks", nil)
			if tt.api.listErr == nil {
				text = strings.TrimPrefix(strings.TrimPrefix(text, "network n1: "), "network n2: ")
			}
			if !res.IsError || text != tt.wantErr {
				t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
			}
			if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "is_error=true") {
				t.Errorf("want Warn log, got %s", logs)
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListNetworksFanOutManyNetworks(t *testing.T) {
	t.Parallel()
	f := &fakeNetworkAPI{routers: map[string][]netbird.NetworkRouter{}}
	for i := range 20 {
		id := fmt.Sprintf("n%d", i)
		f.networks = append(f.networks, netbird.Network{ID: id, Routers: []string{"r-" + id}})
		f.routers[id] = []netbird.NetworkRouter{{ID: "r-" + id}}
	}
	tool, h := listNetworks(fakeClient{networkAPI: f, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	_, out, err := h(t.Context(), nil, listNetworksInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if tool.Name != "list_networks" || len(out.Networks) != 20 {
		t.Fatalf("name=%q networks=%d", tool.Name, len(out.Networks))
	}
	for i, n := range out.Networks {
		if n.ID != f.networks[i].ID || len(n.Routers) != 1 || n.Routers[0].ID != "r-"+n.ID {
			t.Errorf("network %d = %+v", i, n)
		}
	}
}

func TestListNetworksToolDefinition(t *testing.T) {
	t.Parallel()
	tool, _ := listNetworks(fakeClient{networkAPI: &fakeNetworkAPI{}})
	if tool.Name != "list_networks" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Lists all NetBird networks with their routers (routing peer or peer groups, metric, masquerade) and resources (address, type, groups). Disabled routers and resources are included with their enabled flag. Group and peer references include names; names_error appears when names cannot be fetched."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
	cs, _ := connectFake(t, fakeClient{networkAPI: &fakeNetworkAPI{}, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	list, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, lt := range list.Tools {
		if lt.Name != "list_networks" {
			continue
		}
		in, err := json.Marshal(lt.InputSchema)
		if err != nil {
			t.Fatalf("marshal schema: %v", err)
		}
		if strings.Contains(string(in), `"properties":{"`) {
			t.Errorf("input schema has properties: %s", in)
		}
		return
	}
	t.Error("list_networks not registered")
}

func TestListNetworksToolErrorNamesNetwork(t *testing.T) {
	t.Parallel()
	f := &fakeNetworkAPI{
		networks:   []netbird.Network{{ID: "net-xyz", Routers: []string{"r"}}},
		routersErr: &netbird.APIError{Status: 503, Message: "down"},
	}
	cs, _ := connectFake(t, fakeClient{networkAPI: f, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	res, text := callTool(t, cs, "list_networks", nil)
	if !res.IsError || !strings.Contains(text, "net-xyz") {
		t.Errorf("IsError=%v text=%q, want network ID", res.IsError, text)
	}
	f2 := &fakeNetworkAPI{
		networks:     []netbird.Network{{ID: "net-abc", Resources: []string{"x"}}},
		resourcesErr: &netbird.APIError{Status: 503, Message: "down"},
	}
	cs2, _ := connectFake(t, fakeClient{networkAPI: f2, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	res, text = callTool(t, cs2, "list_networks", nil)
	if !res.IsError || !strings.Contains(text, "net-abc") {
		t.Errorf("IsError=%v text=%q, want network ID", res.IsError, text)
	}
}

func TestListNetworksSkipsEmptySubCalls(t *testing.T) {
	t.Parallel()
	f := &fakeNetworkAPI{
		networks: []netbird.Network{
			{ID: "both", Routers: []string{"r1"}, Resources: []string{"x1"}},
			{ID: "routers-only", Routers: []string{"r2"}},
			{ID: "resources-only", Resources: []string{"x3"}},
			{ID: "none"},
		},
		routers:   map[string][]netbird.NetworkRouter{"both": {{ID: "r1"}}, "routers-only": {{ID: "r2"}}},
		resources: map[string][]netbird.NetworkResource{"both": {{ID: "x1"}}, "resources-only": {{ID: "x3"}}},
	}
	_, h := listNetworks(fakeClient{networkAPI: f, groupAPI: &fakeGroupAPI{}, peerAPI: &fakePeerAPI{}})
	_, out, err := h(t.Context(), nil, listNetworksInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	wantRouters := map[string]int{"both": 1, "routers-only": 1}
	wantResources := map[string]int{"both": 1, "resources-only": 1}
	if !reflect.DeepEqual(f.routerIDs, wantRouters) {
		t.Errorf("router calls = %v, want %v", f.routerIDs, wantRouters)
	}
	if !reflect.DeepEqual(f.resourceIDs, wantResources) {
		t.Errorf("resource calls = %v, want %v", f.resourceIDs, wantResources)
	}
	b, err := json.Marshal(out.Networks[3])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"routers":[]`) || !strings.Contains(string(b), `"resources":[]`) {
		t.Errorf("none = %s", b)
	}
}

func TestListNetworksToolDegrades(t *testing.T) {
	t.Parallel()
	newAPI := func() *fakeNetworkAPI {
		return &fakeNetworkAPI{
			networks: []netbird.Network{{ID: "n1", Name: "a", Routers: []string{"r1", "r2"}}},
			routers:  map[string][]netbird.NetworkRouter{"n1": {{ID: "r1", Peer: "p1"}, {ID: "r2", PeerGroups: []string{"g1"}}}},
		}
	}
	boom := &netbird.APIError{Status: 500, Message: "boom"}
	const boomMsg = "netbird api: 500: boom (hint: NetBird server error, retry later)"
	groups := []netbird.Group{{ID: "g1", Name: "devs"}}
	peers := []netbird.Peer{{ID: "p1", Name: "laptop"}}
	const head = `{"networks":[{"id":"n1","name":"a","routing_peers_count":0,"policies":[],"resources":[],"routers":[`
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
			want: head + `{"id":"r1","peer":{"id":"p1","name":"laptop"},"peer_groups":[],"metric":0,"masquerade":false,"enabled":false},` +
				`{"id":"r2","peer_groups":[{"id":"g1"}],"metric":0,"masquerade":false,"enabled":false}]}],"names_error":"` + boomMsg + `"}`,
		},
		{
			name: "peers failure",
			grp:  &fakeGroupAPI{groups: groups},
			peer: &fakePeerAPI{listErr: boom},
			want: head + `{"id":"r1","peer":{"id":"p1"},"peer_groups":[],"metric":0,"masquerade":false,"enabled":false},` +
				`{"id":"r2","peer_groups":[{"id":"g1","name":"devs"}],"metric":0,"masquerade":false,"enabled":false}]}],"names_error":"` + boomMsg + `"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{networkAPI: newAPI(), groupAPI: tt.grp, peerAPI: tt.peer})
			res, text := callTool(t, cs, "list_networks", nil)
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
