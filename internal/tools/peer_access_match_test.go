package tools

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func testPeer() netbird.Peer {
	return netbird.Peer{
		ID: "p1", Name: "laptop", UserID: "u1",
		Groups: []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "all"}},
	}
}

func grp(id, name string) netbird.Group { return netbird.Group{ID: id, Name: name} }

func peerRes(id string) *netbird.Resource { return &netbird.Resource{ID: id, Type: "peer"} }

func TestMatchPolicies(t *testing.T) {
	other := []netbird.Group{grp("g9", "servers")}
	tests := []struct {
		name     string
		policies []netbird.Policy
		want     []accessPolicyOutput
	}{
		{name: "nil policies", policies: nil, want: []accessPolicyOutput{}},
		{
			name: "no match",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: []netbird.Group{grp("g8", "x")}, Destinations: other},
			}}},
			want: []accessPolicyOutput{},
		},
		{
			name: "source group",
			policies: []netbird.Policy{{ID: "a", Name: "A", Enabled: true, Rules: []netbird.PolicyRule{
				{Name: "r", Enabled: true, Action: "accept", Protocol: "tcp", Ports: []string{"22"},
					PortRanges: []netbird.RulePortRange{{Start: 1, End: 2}},
					Sources:    []netbird.Group{grp("g1", "devs")}, Destinations: other,
					DestinationResource: &netbird.Resource{ID: "d1", Type: "host"}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Name: "A", Enabled: true, Rules: []accessRuleOutput{{
				Name: "r", Enabled: true, Action: "accept", Protocol: "tcp", Ports: []string{"22"},
				PortRanges: []portRangeOutput{{Start: 1, End: 2}}, Direction: "source",
				OtherSide:          []ruleGroupOutput{{ID: "g9", Name: "servers"}},
				OtherSideResources: []ruleResourceOutput{{ID: "d1", Type: "host"}},
			}}}},
		},
		{
			name: "destination group",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: other, Destinations: []netbird.Group{grp("g2", "all")},
					SourceResource: &netbird.Resource{ID: "s1", Type: "domain"}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "destination", OtherSide: []ruleGroupOutput{{ID: "g9", Name: "servers"}},
				OtherSideResources: []ruleResourceOutput{{ID: "s1", Type: "domain"}},
			}}}},
		},
		{
			name: "source resource peer",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{SourceResource: peerRes("p1"), Destinations: other},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "source", OtherSide: []ruleGroupOutput{{ID: "g9", Name: "servers"}},
			}}}},
		},
		{
			name: "destination resource peer",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: other, DestinationResource: peerRes("p1")},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "destination", OtherSide: []ruleGroupOutput{{ID: "g9", Name: "servers"}},
			}}}},
		},
		{
			name: "resource of another peer or non-peer type does not match",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{SourceResource: peerRes("p2"), DestinationResource: &netbird.Resource{ID: "p1", Type: "host"}},
			}}},
			want: []accessPolicyOutput{},
		},
		{
			name: "bidirectional source side gives both with opposite side",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Bidirectional: true, Sources: []netbird.Group{grp("g1", "devs")}, Destinations: other},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "both", OtherSide: []ruleGroupOutput{{ID: "g9", Name: "servers"}},
			}}}},
		},
		{
			name: "bidirectional destination side gives both with sources",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Bidirectional: true, Sources: other, Destinations: []netbird.Group{grp("g1", "devs")}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "both", OtherSide: []ruleGroupOutput{{ID: "g9", Name: "servers"}},
			}}}},
		},
		{
			name: "peer on both sides merges and dedupes other side",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: []netbird.Group{grp("g1", "devs"), grp("g5", "five")},
					Destinations: []netbird.Group{grp("g2", "all"), grp("g5", "five")}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "both",
				OtherSide: []ruleGroupOutput{{ID: "g1", Name: "devs"}, {ID: "g5", Name: "five"}, {ID: "g2", Name: "all"}},
			}}}},
		},
		{
			name: "both sides with exactly one resource keeps it",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: []netbird.Group{grp("g1", "devs")}, Destinations: []netbird.Group{grp("g2", "all")},
					DestinationResource: &netbird.Resource{ID: "d1", Type: "host"}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction:          "both",
				OtherSide:          []ruleGroupOutput{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "all"}},
				OtherSideResources: []ruleResourceOutput{{ID: "d1", Type: "host"}},
			}}}},
		},
		{
			name: "both sides with two resources keeps both in source then destination order",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{Sources: []netbird.Group{grp("g1", "devs")}, DestinationResource: peerRes("p1"),
					SourceResource: &netbird.Resource{ID: "s1", Type: "host"}},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "both", OtherSide: []ruleGroupOutput{{ID: "g1", Name: "devs"}},
				OtherSideResources: []ruleResourceOutput{{ID: "s1", Type: "host"}, {ID: "p1", Type: "peer"}},
			}}}},
		},
		{
			name: "same resource on both sides is deduplicated",
			policies: []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{
				{SourceResource: peerRes("p1"), DestinationResource: peerRes("p1")},
			}}},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Direction: "both", OtherSide: []ruleGroupOutput{},
				OtherSideResources: []ruleResourceOutput{{ID: "p1", Type: "peer"}},
			}}}},
		},
		{
			name: "non-matching rules dropped and policy without matches dropped",
			policies: []netbird.Policy{
				{ID: "a", Rules: []netbird.PolicyRule{
					{Name: "no", Sources: []netbird.Group{grp("g8", "x")}},
					{Name: "yes", Sources: []netbird.Group{grp("g1", "devs")}},
				}},
				{ID: "b", Rules: []netbird.PolicyRule{{Name: "no", Sources: []netbird.Group{grp("g8", "x")}}}},
			},
			want: []accessPolicyOutput{{ID: "a", Rules: []accessRuleOutput{{
				Name: "yes", Direction: "source", OtherSide: []ruleGroupOutput{},
			}}}},
		},
		{
			name: "disabled policy and rule kept with flag, API order kept",
			policies: []netbird.Policy{
				{ID: "z", Enabled: false, Rules: []netbird.PolicyRule{{Enabled: false, Sources: []netbird.Group{grp("g1", "devs")}}}},
				{ID: "a", Enabled: true, Rules: []netbird.PolicyRule{{Enabled: true, Destinations: []netbird.Group{grp("g2", "all")}}}},
			},
			want: []accessPolicyOutput{
				{ID: "z", Rules: []accessRuleOutput{{Direction: "source", OtherSide: []ruleGroupOutput{}}}},
				{ID: "a", Enabled: true, Rules: []accessRuleOutput{{Enabled: true, Direction: "destination", OtherSide: []ruleGroupOutput{}}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPolicies(testPeer(), tt.policies)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchPolicies:\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestMatchRoutes(t *testing.T) {
	tests := []struct {
		name   string
		routes []netbird.Route
		want   []accessRouteOutput
	}{
		{name: "nil routes", routes: nil, want: []accessRouteOutput{}},
		{
			name:   "routing peer by id",
			routes: []netbird.Route{{ID: "r", NetworkID: "n", Network: "10.0.0.0/8", Enabled: true, Peer: "p1"}},
			want: []accessRouteOutput{{ID: "r", NetworkID: "n", Network: "10.0.0.0/8", Domains: []string{}, Enabled: true,
				Roles: []string{"routing_peer"}}},
		},
		{
			name:   "routing peer by peer group",
			routes: []netbird.Route{{ID: "r", PeerGroups: []string{"g8", "g1"}}},
			want:   []accessRouteOutput{{ID: "r", Domains: []string{}, Roles: []string{"routing_peer"}}},
		},
		{
			name:   "in_groups",
			routes: []netbird.Route{{ID: "r", Groups: []string{"g2"}, Domains: []string{"a.example"}}},
			want:   []accessRouteOutput{{ID: "r", Domains: []string{"a.example"}, Roles: []string{"in_groups"}}},
		},
		{
			name:   "access_control",
			routes: []netbird.Route{{ID: "r", AccessControlGroups: []string{"g1"}}},
			want:   []accessRouteOutput{{ID: "r", Domains: []string{}, Roles: []string{"access_control"}}},
		},
		{
			name: "multiple roles in fixed order",
			routes: []netbird.Route{{ID: "r", AccessControlGroups: []string{"g1"}, Groups: []string{"g1"},
				Peer: "p1"}},
			want: []accessRouteOutput{{ID: "r", Domains: []string{},
				Roles: []string{"routing_peer", "in_groups", "access_control"}}},
		},
		{
			name: "no role dropped, order kept",
			routes: []netbird.Route{
				{ID: "z", Peer: "p1"}, {ID: "x", Peer: "p2", Groups: []string{"g8"}}, {ID: "a", Groups: []string{"g1"}},
			},
			want: []accessRouteOutput{
				{ID: "z", Domains: []string{}, Roles: []string{"routing_peer"}},
				{ID: "a", Domains: []string{}, Roles: []string{"in_groups"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRoutes(testPeer(), tt.routes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchRoutes:\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestToPeerAccessOutput(t *testing.T) {
	peer := testPeer()
	got := toPeerAccessOutput(peer, nil, nil)
	if got.Peer.ID != "p1" || got.Peer.Name != "laptop" {
		t.Errorf("peer detail not mapped: %+v", got.Peer)
	}
	if got.Peer.UserID != "u1" {
		t.Errorf("peer user_id = %q, want u1", got.Peer.UserID)
	}
	if len(got.Peer.Groups) != 2 || got.Peer.Groups[0].ID != "g1" || got.Peer.Groups[1].ID != "g2" {
		t.Errorf("peer groups = %+v", got.Peer.Groups)
	}

	policies := []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{{Sources: []netbird.Group{grp("g1", "devs")}}}}}
	routes := []netbird.Route{{ID: "r", Peer: "p1"}}
	got = toPeerAccessOutput(peer, policies, routes)
	if len(got.Policies) != 1 || got.Policies[0].ID != "a" || len(got.Routes) != 1 || got.Routes[0].ID != "r" {
		t.Errorf("policies/routes not joined: %+v %+v", got.Policies, got.Routes)
	}
}

func TestPeerAccessOutputJSON(t *testing.T) {
	t.Run("other_side_resources encodes as a list", func(t *testing.T) {
		policies := []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{{
			Sources: []netbird.Group{grp("g1", "devs")}, DestinationResource: &netbird.Resource{ID: "d1", Type: "host"},
		}}}}
		b, err := json.Marshal(toPeerAccessOutput(testPeer(), policies, nil))
		if err != nil {
			t.Fatal(err)
		}
		if want := `"other_side_resources":[{"id":"d1","type":"host"}]`; !strings.Contains(string(b), want) {
			t.Errorf("output missing %s: %s", want, b)
		}
		if strings.Contains(string(b), `"other_side_resource"`) {
			t.Errorf("output contains stale key other_side_resource: %s", b)
		}
	})

	t.Run("empty lists encode as [] not null", func(t *testing.T) {
		b, err := json.Marshal(toPeerAccessOutput(netbird.Peer{ID: "p1"}, nil, nil))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"policies", "routes"} {
			if string(m[k]) != "[]" {
				t.Errorf("%s = %s, want []", k, m[k])
			}
		}
	})

	t.Run("optional keys omitted and other_side always present", func(t *testing.T) {
		policies := []netbird.Policy{{ID: "a", Rules: []netbird.PolicyRule{{Sources: []netbird.Group{grp("g1", "devs")}}}}}
		routes := []netbird.Route{{ID: "r", Peer: "p1"}}
		b, err := json.Marshal(toPeerAccessOutput(testPeer(), policies, routes))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, bad := range []string{`"ports"`, `"port_ranges"`, `"other_side_resources"`, `"other_side_resource"`, `"network":`, `null`} {
			if strings.Contains(s, bad) {
				t.Errorf("output contains %s: %s", bad, s)
			}
		}
		for _, want := range []string{`"other_side":[]`, `"domains":[]`, `"roles":["routing_peer"]`} {
			if !strings.Contains(s, want) {
				t.Errorf("output missing %s: %s", want, s)
			}
		}
	})
}
