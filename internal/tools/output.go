package tools

import "github.com/koinsaari/netbird-mcp/internal/netbird"

type refOutput struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func toGroupOutputs(groups []netbird.Group) []refOutput {
	out := make([]refOutput, 0, len(groups))
	for _, g := range groups {
		out = append(out, refOutput{ID: g.ID, Name: g.Name})
	}
	return out
}

func groupNamesByID(groups []netbird.Group) map[string]string {
	names := make(map[string]string, len(groups))
	for _, g := range groups {
		names[g.ID] = g.Name
	}
	return names
}

func toGroupRefs(ids []string, names map[string]string) []refOutput {
	out := make([]refOutput, 0, len(ids))
	for _, id := range ids {
		out = append(out, refOutput{ID: id, Name: names[id]})
	}
	return out
}

func peerNamesByID(peers []netbird.Peer) map[string]string {
	names := make(map[string]string, len(peers))
	for _, p := range peers {
		names[p.ID] = p.Name
	}
	return names
}

func toPeerRef(id string, names map[string]string) *refOutput {
	if id == "" {
		return nil
	}
	return &refOutput{ID: id, Name: names[id]}
}

type localFlagsOutput struct {
	RosenpassEnabled      *bool `json:"rosenpass_enabled,omitempty"`
	RosenpassPermissive   *bool `json:"rosenpass_permissive,omitempty"`
	ServerSSHAllowed      *bool `json:"server_ssh_allowed,omitempty"`
	RemoteJobsAllowed     *bool `json:"remote_jobs_allowed,omitempty"`
	DisableClientRoutes   *bool `json:"disable_client_routes,omitempty"`
	DisableServerRoutes   *bool `json:"disable_server_routes,omitempty"`
	DisableDNS            *bool `json:"disable_dns,omitempty"`
	DisableFirewall       *bool `json:"disable_firewall,omitempty"`
	BlockLANAccess        *bool `json:"block_lan_access,omitempty"`
	BlockInbound          *bool `json:"block_inbound,omitempty"`
	LazyConnectionEnabled *bool `json:"lazy_connection_enabled,omitempty"`
}

func toLocalFlagsOutput(f *netbird.PeerLocalFlags) *localFlagsOutput {
	if f == nil {
		return nil
	}
	o := localFlagsOutput(*f)
	return &o
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
