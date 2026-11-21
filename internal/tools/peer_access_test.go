package tools

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

const accessPeerID = "cdl0ajd5jtbs73abcdef"
const barrierTimeout = 2 * time.Second

type accessFakes struct {
	peer     netbird.Peer
	policies []netbird.Policy
	routes   []netbird.Route

	peerErr, policyErr, routeErr error

	arrived    atomic.Int32
	all        chan struct{}
	once       sync.Once
	peerCalls  atomic.Int32
	polCalls   atomic.Int32
	routeCalls atomic.Int32
	cancelled  sync.Map
	peerID     atomic.Value
}

func newAccessFakes() *accessFakes { return &accessFakes{all: make(chan struct{})} }

func (f *accessFakes) failing() bool {
	return f.peerErr != nil || f.policyErr != nil || f.routeErr != nil
}

func (f *accessFakes) call(ctx context.Context, name string, own error) error {
	if f.arrived.Add(1) == 3 {
		f.once.Do(func() { close(f.all) })
	}
	select {
	case <-f.all:
	case <-time.After(barrierTimeout):
		return errors.New("barrier timeout: calls are not concurrent")
	case <-ctx.Done():
		f.cancelled.Store(name, struct{}{})
		return ctx.Err()
	}
	if own != nil {
		return own
	}
	if f.failing() {
		select {
		case <-ctx.Done():
			f.cancelled.Store(name, struct{}{})
			return ctx.Err()
		case <-time.After(barrierTimeout):
			return errors.New("sibling was not cancelled")
		}
	}
	return nil
}

func (f *accessFakes) sawCancel(name string) bool {
	_, ok := f.cancelled.Load(name)
	return ok
}

type accessPeerFake struct {
	peerAPI
	f *accessFakes
}

func (a accessPeerFake) GetPeer(ctx context.Context, id string) (netbird.Peer, error) {
	a.f.peerCalls.Add(1)
	a.f.peerID.Store(id)
	if err := a.f.call(ctx, "peer", a.f.peerErr); err != nil {
		return netbird.Peer{}, err
	}
	return a.f.peer, nil
}

type accessPolicyFake struct{ f *accessFakes }

func (a accessPolicyFake) ListPolicies(ctx context.Context) ([]netbird.Policy, error) {
	a.f.polCalls.Add(1)
	if err := a.f.call(ctx, "policies", a.f.policyErr); err != nil {
		return nil, err
	}
	return a.f.policies, nil
}

type accessRouteFake struct{ f *accessFakes }

func (a accessRouteFake) ListRoutes(ctx context.Context) ([]netbird.Route, error) {
	a.f.routeCalls.Add(1)
	if err := a.f.call(ctx, "routes", a.f.routeErr); err != nil {
		return nil, err
	}
	return a.f.routes, nil
}

func (f *accessFakes) api() peerAccessAPI {
	return struct {
		peerAPI
		policyAPI
		routeAPI
	}{accessPeerFake{f: f}, accessPolicyFake{f}, accessRouteFake{f}}
}

func (f *accessFakes) client() Client {
	return fakeClient{peerAPI: accessPeerFake{f: f}, policyAPI: accessPolicyFake{f}, routeAPI: accessRouteFake{f}}
}

func TestGetPeerAccessTool(t *testing.T) {
	t.Parallel()
	f := newAccessFakes()
	f.peer = netbird.Peer{
		ID: accessPeerID, Name: "laptop", UserID: "u1", OS: "linux",
		Groups: []netbird.Group{{ID: "g1", Name: "devs", PeersCount: 4}},
	}
	f.policies = []netbird.Policy{
		{ID: "p1", Name: "ssh", Enabled: true, Rules: []netbird.PolicyRule{{
			Name: "r1", Enabled: true, Action: "accept", Protocol: "tcp", Ports: []string{"22"},
			Sources:      []netbird.Group{{ID: "g1", Name: "devs"}},
			Destinations: []netbird.Group{{ID: "g2", Name: "servers"}},
		}}},
		{ID: "p2", Name: "other", Enabled: true, Rules: []netbird.PolicyRule{{
			Name: "r2", Sources: []netbird.Group{{ID: "gx"}}, Destinations: []netbird.Group{{ID: "gy"}},
		}}},
	}
	f.routes = []netbird.Route{
		{ID: "r1", NetworkID: "lan", Network: "10.0.0.0/24", Enabled: true, Peer: accessPeerID},
		{ID: "r2", NetworkID: "other", Peer: "nope"},
	}

	cs, logs := connectFake(t, f.client())
	res, text := callTool(t, cs, "get_peer_access", map[string]any{"peer_id": accessPeerID})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	sc, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content = %#v", res.StructuredContent)
	}
	if peer, _ := sc["peer"].(map[string]any); peer["id"] != accessPeerID || peer["name"] != "laptop" {
		t.Errorf("peer = %#v", sc["peer"])
	}
	peer, _ := sc["peer"].(map[string]any)
	wantKeys := []string{"connected", "dns_label", "groups", "id", "ip", "name"}
	var gotKeys []string
	for k := range peer {
		gotKeys = append(gotKeys, k)
	}
	slices.Sort(gotKeys)
	if !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("peer keys = %v, want %v", gotKeys, wantKeys)
	}
	if g, _ := peer["groups"].([]any); len(g) == 1 {
		if gm, _ := g[0].(map[string]any); len(gm) != 2 {
			t.Errorf("group keys = %#v, want id and name only", gm)
		}
	}
	if g, _ := peer["groups"].([]any); len(g) != 1 || g[0].(map[string]any)["id"] != "g1" {
		t.Errorf("peer groups = %#v", peer["groups"])
	}
	for _, dup := range []string{"groups", "user_id", "os"} {
		if _, has := sc[dup]; has {
			t.Errorf("top-level %q repeats the peer object", dup)
		}
	}
	pols, _ := sc["policies"].([]any)
	if len(pols) != 1 || pols[0].(map[string]any)["id"] != "p1" {
		t.Fatalf("policies = %#v", sc["policies"])
	}
	rules, _ := pols[0].(map[string]any)["rules"].([]any)
	if len(rules) != 1 || rules[0].(map[string]any)["direction"] != "source" {
		t.Errorf("rules = %#v", rules)
	}
	routes, _ := sc["routes"].([]any)
	if len(routes) != 1 || routes[0].(map[string]any)["id"] != "r1" {
		t.Fatalf("routes = %#v", sc["routes"])
	}
	if roles, _ := routes[0].(map[string]any)["roles"].([]any); len(roles) != 1 || roles[0] != "routing_peer" {
		t.Errorf("roles = %#v", roles)
	}
	if got, _ := f.peerID.Load().(string); got != accessPeerID {
		t.Errorf("peer id = %q", got)
	}
	if f.peerCalls.Load() != 1 || f.polCalls.Load() != 1 || f.routeCalls.Load() != 1 {
		t.Errorf("calls = %d/%d/%d, want 1 each", f.peerCalls.Load(), f.polCalls.Load(), f.routeCalls.Load())
	}
	if !strings.Contains(logs.String(), "tool=get_peer_access") || !strings.Contains(logs.String(), "is_error=false") {
		t.Errorf("logs missing tool fields: %s", logs)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestGetPeerAccessToolInvalidID(t *testing.T) {
	for _, id := range []string{"", accessPeerID[:19], "CDL0AJD5JTBS73ABCDEF", "../x", "cdl0ajd5jtbs73abcdew"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			f := newAccessFakes()
			cs, _ := connectFake(t, f.client())
			res, text := callTool(t, cs, "get_peer_access", map[string]any{"peer_id": id})
			if !res.IsError || text != errInvalidPeerID.Error() {
				t.Errorf("IsError=%v text=%q, want %q", res.IsError, text, errInvalidPeerID)
			}
			if n := f.peerCalls.Load() + f.polCalls.Load() + f.routeCalls.Load(); n != 0 {
				t.Errorf("API calls = %d, want 0", n)
			}
		})
	}
}

func TestGetPeerAccessToolRunsConcurrently(t *testing.T) {
	t.Parallel()
	f := newAccessFakes()
	_, h := getPeerAccess(f.api())
	start := time.Now()
	_, _, err := h(t.Context(), nil, getPeerAccessInput{PeerID: accessPeerID})
	if err != nil {
		t.Fatalf("handler error (sequential?): %v", err)
	}
	if d := time.Since(start); d >= barrierTimeout {
		t.Errorf("took %v, want calls to overlap", d)
	}
}

func TestGetPeerAccessToolFailure(t *testing.T) {
	unauthorized := &netbird.APIError{Status: 401, Message: "bad token"}
	notFound := &netbird.APIError{Status: 404, Message: "peer not found"}
	tests := []struct {
		name       string
		set        func(f *accessFakes)
		wantErr    string
		wantCancel []string
	}{
		{
			name: "peer 404", set: func(f *accessFakes) { f.peerErr = notFound },
			wantErr:    "netbird api: 404: peer not found (hint: not found; check the ID or NETBIRD_API_URL)",
			wantCancel: []string{"policies", "routes"},
		},
		{
			name: "policies 401", set: func(f *accessFakes) { f.policyErr = unauthorized },
			wantErr:    "netbird api: 401: bad token (hint: token invalid or expired)",
			wantCancel: []string{"peer", "routes"},
		},
		{
			name: "routes 401", set: func(f *accessFakes) { f.routeErr = unauthorized },
			wantErr:    "netbird api: 401: bad token (hint: token invalid or expired)",
			wantCancel: []string{"peer", "policies"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAccessFakes()
			tt.set(f)
			cs, logs := connectFake(t, f.client())
			res, text := callTool(t, cs, "get_peer_access", map[string]any{"peer_id": accessPeerID})
			if !res.IsError || text != tt.wantErr {
				t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
			}
			for _, name := range tt.wantCancel {
				if !f.sawCancel(name) {
					t.Errorf("%s call did not see a cancelled context", name)
				}
			}
			if !strings.Contains(logs.String(), "is_error=true") {
				t.Errorf("logs missing is_error=true: %s", logs)
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestGetPeerAccessToolReturnsOriginalError(t *testing.T) {
	t.Parallel()
	want := &netbird.APIError{Status: 500, Message: "boom"}
	f := newAccessFakes()
	f.policyErr = want
	_, h := getPeerAccess(f.api())
	_, _, err := h(t.Context(), nil, getPeerAccessInput{PeerID: accessPeerID})
	if !errors.Is(err, want) || errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the original error and not context.Canceled", err)
	}
}

func TestGetPeerAccessToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := getPeerAccess(newAccessFakes().api())
	if tool.Name != "get_peer_access" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Describes one NetBird peer's access: the access policies whose rules apply to it (with direction and the other side) and the network routes it is part of. Disabled policies, rules and routes are included with their enabled flag. Posture checks are not evaluated."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
