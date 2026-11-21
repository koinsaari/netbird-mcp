package tools

import "github.com/koinsaari/netbird-mcp/internal/netbird"

type groupOutput struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func toGroupOutputs(groups []netbird.Group) []groupOutput {
	out := make([]groupOutput, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupOutput{ID: g.ID, Name: g.Name})
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

func toGroupRefs(ids []string, names map[string]string) []groupOutput {
	out := make([]groupOutput, 0, len(ids))
	for _, id := range ids {
		out = append(out, groupOutput{ID: id, Name: names[id]})
	}
	return out
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
