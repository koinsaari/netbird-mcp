package tools

import (
	"slices"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type peerAccessOutput struct {
	Peer          accessPeerOutput      `json:"peer"`
	Policies      []accessPolicyOutput  `json:"policies"`
	Routes        []accessRouteOutput   `json:"routes"`
	Networks      []accessNetworkOutput `json:"networks"`
	NetworksError string                `json:"networks_error,omitempty"`
}

type accessNetworkOutput struct {
	ID        string                        `json:"id"`
	Name      string                        `json:"name"`
	Routers   []accessNetworkRouterOutput   `json:"routers"`
	Resources []accessNetworkResourceOutput `json:"resources"`
}

type accessNetworkRouterOutput struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type accessNetworkResourceOutput struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	Type        string   `json:"type"`
	Enabled     bool     `json:"enabled"`
	ViaPolicies []string `json:"via_policies"`
}

type accessPeerOutput struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	IP         string            `json:"ip"`
	DNSLabel   string            `json:"dns_label"`
	Connected  bool              `json:"connected"`
	Groups     []refOutput       `json:"groups"`
	LocalFlags *localFlagsOutput `json:"local_flags,omitempty"`
}

type accessPolicyOutput struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Enabled bool               `json:"enabled"`
	Rules   []accessRuleOutput `json:"rules"`
}

type accessRuleOutput struct {
	Name               string               `json:"name"`
	Enabled            bool                 `json:"enabled"`
	Action             string               `json:"action"`
	Protocol           string               `json:"protocol"`
	Ports              []string             `json:"ports,omitempty"`
	PortRanges         []portRangeOutput    `json:"port_ranges,omitempty"`
	Direction          string               `json:"direction"`
	OtherSide          []ruleGroupOutput    `json:"other_side"`
	OtherSideResources []ruleResourceOutput `json:"other_side_resources,omitempty"`
}

type accessRouteOutput struct {
	ID        string   `json:"id"`
	NetworkID string   `json:"network_id"`
	Network   string   `json:"network,omitempty"`
	Domains   []string `json:"domains"`
	Enabled   bool     `json:"enabled"`
	Roles     []string `json:"roles"`
}

func matchPolicies(peer netbird.Peer, policies []netbird.Policy) []accessPolicyOutput {
	groupIDs := peerGroupIDs(peer)
	out := make([]accessPolicyOutput, 0, len(policies))
	for _, p := range policies {
		rules := make([]accessRuleOutput, 0, len(p.Rules))
		for _, r := range p.Rules {
			if d, ok := accessRule(peer.ID, groupIDs, r); ok {
				rules = append(rules, d)
			}
		}
		if len(rules) == 0 {
			continue
		}
		out = append(out, accessPolicyOutput{ID: p.ID, Name: p.Name, Enabled: p.Enabled, Rules: rules})
	}
	return out
}

func accessRule(peerID string, groupIDs map[string]bool, r netbird.PolicyRule) (accessRuleOutput, bool) {
	onSource := anyGroup(groupIDs, r.Sources) || isPeerResource(peerID, r.SourceResource)
	onDest := anyGroup(groupIDs, r.Destinations) || isPeerResource(peerID, r.DestinationResource)
	if !onSource && !onDest {
		return accessRuleOutput{}, false
	}

	d := accessRuleOutput{
		Name: r.Name, Enabled: r.Enabled, Action: r.Action, Protocol: r.Protocol, Ports: r.Ports,
	}
	for _, pr := range r.PortRanges {
		d.PortRanges = append(d.PortRanges, portRangeOutput{Start: pr.Start, End: pr.End})
	}

	switch {
	case onSource && onDest:
		d.Direction = "both"
		d.OtherSide = mergeGroups(r.Sources, r.Destinations)
		d.OtherSideResources = mergeResources(r.SourceResource, r.DestinationResource)
	case onSource:
		d.Direction = "source"
		d.OtherSide = toRuleGroups(r.Destinations)
		d.OtherSideResources = mergeResources(r.DestinationResource)
	default:
		d.Direction = "destination"
		d.OtherSide = toRuleGroups(r.Sources)
		d.OtherSideResources = mergeResources(r.SourceResource)
	}
	if r.Bidirectional {
		d.Direction = "both"
	}
	return d, true
}

func peerGroupIDs(peer netbird.Peer) map[string]bool {
	ids := make(map[string]bool, len(peer.Groups))
	for _, g := range peer.Groups {
		ids[g.ID] = true
	}
	return ids
}

func anyGroup(ids map[string]bool, groups []netbird.Group) bool {
	for _, g := range groups {
		if ids[g.ID] {
			return true
		}
	}
	return false
}

func anyID(ids map[string]bool, list []string) bool {
	for _, id := range list {
		if ids[id] {
			return true
		}
	}
	return false
}

func isPeerResource(peerID string, r *netbird.Resource) bool {
	return r != nil && r.Type == "peer" && r.ID == peerID
}

func isNetworkResource(resID string, r *netbird.Resource) bool {
	return r != nil && r.Type != "peer" && r.ID == resID
}

func mergeGroups(a, b []netbird.Group) []ruleGroupOutput {
	out := make([]ruleGroupOutput, 0, len(a)+len(b))
	seen := make(map[string]bool, len(a)+len(b))
	for _, g := range append(append([]netbird.Group{}, a...), b...) {
		if seen[g.ID] {
			continue
		}
		seen[g.ID] = true
		out = append(out, ruleGroupOutput{ID: g.ID, Name: g.Name})
	}
	return out
}

func mergeResources(resources ...*netbird.Resource) []ruleResourceOutput {
	var out []ruleResourceOutput
	for _, r := range resources {
		if r == nil || slices.Contains(out, ruleResourceOutput{ID: r.ID, Type: r.Type}) {
			continue
		}
		out = append(out, ruleResourceOutput{ID: r.ID, Type: r.Type})
	}
	return out
}

func matchRoutes(peer netbird.Peer, routes []netbird.Route) []accessRouteOutput {
	groupIDs := peerGroupIDs(peer)
	out := make([]accessRouteOutput, 0, len(routes))
	for _, r := range routes {
		roles := []string{}
		if (r.Peer != "" && r.Peer == peer.ID) || anyID(groupIDs, r.PeerGroups) {
			roles = append(roles, "routing_peer")
		}
		if anyID(groupIDs, r.Groups) {
			roles = append(roles, "in_groups")
		}
		if anyID(groupIDs, r.AccessControlGroups) {
			roles = append(roles, "access_control")
		}
		if len(roles) == 0 {
			continue
		}
		out = append(out, accessRouteOutput{
			ID: r.ID, NetworkID: r.NetworkID, Network: r.Network, Domains: nonNil(r.Domains),
			Enabled: r.Enabled, Roles: roles,
		})
	}
	return out
}

func matchNetworks(peer netbird.Peer, policies []netbird.Policy, networks []netbird.Network, routers [][]netbird.NetworkRouter, resources [][]netbird.NetworkResource) []accessNetworkOutput {
	groupIDs := peerGroupIDs(peer)
	out := make([]accessNetworkOutput, 0, len(networks))
	for i, n := range networks {
		matched := accessNetworkOutput{
			ID: n.ID, Name: n.Name,
			Routers:   []accessNetworkRouterOutput{},
			Resources: []accessNetworkResourceOutput{},
		}
		for _, r := range routers[i] {
			if (r.Peer != "" && r.Peer == peer.ID) || anyID(groupIDs, r.PeerGroups) {
				matched.Routers = append(matched.Routers, accessNetworkRouterOutput{ID: r.ID, Enabled: r.Enabled})
			}
		}
		for _, res := range resources[i] {
			via := reachingPolicies(peer.ID, groupIDs, policies, res)
			if len(via) == 0 {
				continue
			}
			matched.Resources = append(matched.Resources, accessNetworkResourceOutput{
				ID: res.ID, Name: res.Name, Address: res.Address, Type: res.Type, Enabled: res.Enabled,
				ViaPolicies: via,
			})
		}
		if len(matched.Routers) == 0 && len(matched.Resources) == 0 {
			continue
		}
		out = append(out, matched)
	}
	return out
}

func reachingPolicies(peerID string, groupIDs map[string]bool, policies []netbird.Policy, res netbird.NetworkResource) []string {
	resGroupIDs := make(map[string]bool, len(res.Groups))
	for _, g := range res.Groups {
		resGroupIDs[g.ID] = true
	}
	via := []string{}
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		for _, r := range p.Rules {
			if ruleReaches(peerID, groupIDs, resGroupIDs, res.ID, r) {
				via = append(via, p.ID)
				break
			}
		}
	}
	return via
}

func ruleReaches(peerID string, groupIDs, resGroupIDs map[string]bool, resID string, r netbird.PolicyRule) bool {
	if !r.Enabled || r.Action != "accept" {
		return false
	}
	peerSource := anyGroup(groupIDs, r.Sources) || isPeerResource(peerID, r.SourceResource)
	peerDest := anyGroup(groupIDs, r.Destinations) || isPeerResource(peerID, r.DestinationResource)
	resSource := anyGroup(resGroupIDs, r.Sources) || isNetworkResource(resID, r.SourceResource)
	resDest := anyGroup(resGroupIDs, r.Destinations) || isNetworkResource(resID, r.DestinationResource)
	return (peerSource && resDest) || (r.Bidirectional && peerDest && resSource)
}

func toPeerAccessOutput(peer netbird.Peer, policies []netbird.Policy, routes []netbird.Route, networks []netbird.Network, routers [][]netbird.NetworkRouter, resources [][]netbird.NetworkResource, networksErr string) peerAccessOutput {
	return peerAccessOutput{
		Peer:     toAccessPeerOutput(peer),
		Policies: matchPolicies(peer, policies), Routes: matchRoutes(peer, routes),
		Networks:      matchNetworks(peer, policies, networks, routers, resources),
		NetworksError: networksErr,
	}
}

func toAccessPeerOutput(p netbird.Peer) accessPeerOutput {
	return accessPeerOutput{
		ID: p.ID, Name: p.Name, IP: p.IP, DNSLabel: p.DNSLabel, Connected: p.Connected,
		Groups: toGroupOutputs(p.Groups), LocalFlags: toLocalFlagsOutput(p.LocalFlags),
	}
}
