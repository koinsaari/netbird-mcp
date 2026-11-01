package tools

import (
	"context"
	"errors"
	"net/netip"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

var (
	errInvalidIP     = errors.New("invalid ip: must be an IP address without zone")
	errInvalidPeerID = errors.New("invalid peer_id: must be a 20-character NetBird peer ID")
	peerIDPattern    = regexp.MustCompile(`^[0-9a-v]{20}$`)
)

type peerAPI interface {
	ListPeers(ctx context.Context, params netbird.ListPeersParams) ([]netbird.Peer, error)
	GetPeer(ctx context.Context, id string) (netbird.Peer, error)
}

type listPeersInput struct {
	Name string `json:"name,omitempty" jsonschema:"filter by peer name"`
	IP   string `json:"ip,omitempty" jsonschema:"filter by peer NetBird IP address"`
}

type listPeersOutput struct {
	Peers []peerOutput `json:"peers"`
}

type peerOutput struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Hostname         string        `json:"hostname"`
	DNSLabel         string        `json:"dns_label"`
	IP               string        `json:"ip"`
	Connected        bool          `json:"connected"`
	LastSeen         string        `json:"last_seen"` // RFC 3339 string passed through
	OS               string        `json:"os"`
	Version          string        `json:"version"`
	Groups           []groupOutput `json:"groups"`
	UserID           string        `json:"user_id"`
	LoginExpired     bool          `json:"login_expired"`
	ApprovalRequired bool          `json:"approval_required"`
}

type groupOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listPeers builds the list_peers tool definition and its handler. The handler
// returns raw errors; Register maps them to model-facing messages.
func listPeers(c peerAPI) (*mcp.Tool, mcp.ToolHandlerFor[listPeersInput, listPeersOutput]) {
	tool := &mcp.Tool{
		Name:        "list_peers",
		Description: "Lists peers (devices) in the NetBird account, optionally filtered by name or IP address.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in listPeersInput) (*mcp.CallToolResult, listPeersOutput, error) {
		params := netbird.ListPeersParams{Name: in.Name}
		if in.IP != "" {
			addr, err := netip.ParseAddr(in.IP)
			if err != nil || addr.Zone() != "" {
				return nil, listPeersOutput{}, errInvalidIP
			}
			params.IP = addr.Unmap().String()
		}
		peers, err := c.ListPeers(ctx, params)
		if err != nil {
			return nil, listPeersOutput{}, err
		}
		return nil, toListPeersOutput(peers), nil
	}
	return tool, handler
}

func toListPeersOutput(peers []netbird.Peer) listPeersOutput {
	out := listPeersOutput{Peers: make([]peerOutput, 0, len(peers))}
	for _, p := range peers {
		out.Peers = append(out.Peers, toPeerOutput(p))
	}
	return out
}

func toPeerOutput(p netbird.Peer) peerOutput {
	groups := make([]groupOutput, 0, len(p.Groups))
	for _, g := range p.Groups {
		groups = append(groups, groupOutput{ID: g.ID, Name: g.Name})
	}
	return peerOutput{
		ID: p.ID, Name: p.Name, Hostname: p.Hostname, DNSLabel: p.DNSLabel, IP: p.IP,
		Connected: p.Connected, LastSeen: p.LastSeen, OS: p.OS, Version: p.Version,
		Groups: groups, UserID: p.UserID, LoginExpired: p.LoginExpired,
		ApprovalRequired: p.ApprovalRequired,
	}
}

type getPeerInput struct {
	PeerID string `json:"peer_id" jsonschema:"the 20-character NetBird peer ID, as returned by list_peers"`
}

type peerDetailOutput struct {
	ID                          string                 `json:"id"`
	Name                        string                 `json:"name"`
	CreatedAt                   string                 `json:"created_at"`
	IP                          string                 `json:"ip"`
	IPv6                        string                 `json:"ipv6,omitempty"`
	NetworkAddresses            []networkAddressOutput `json:"network_addresses,omitzero"`
	ConnectionIP                string                 `json:"connection_ip"`
	Connected                   bool                   `json:"connected"`
	LastSeen                    string                 `json:"last_seen"`
	OS                          string                 `json:"os"`
	KernelVersion               string                 `json:"kernel_version"`
	GeonameID                   int                    `json:"geoname_id"`
	Version                     string                 `json:"version"`
	Groups                      []groupDetailOutput    `json:"groups"`
	SSHEnabled                  bool                   `json:"ssh_enabled"`
	UserID                      string                 `json:"user_id"`
	Hostname                    string                 `json:"hostname"`
	UIVersion                   string                 `json:"ui_version"`
	DNSLabel                    string                 `json:"dns_label"`
	LoginExpirationEnabled      bool                   `json:"login_expiration_enabled"`
	LoginExpired                bool                   `json:"login_expired"`
	LastLogin                   string                 `json:"last_login"`
	InactivityExpirationEnabled bool                   `json:"inactivity_expiration_enabled"`
	ApprovalRequired            bool                   `json:"approval_required"`
	DisapprovalReason           string                 `json:"disapproval_reason,omitempty"`
	CountryCode                 string                 `json:"country_code"`
	CityName                    string                 `json:"city_name"`
	SerialNumber                string                 `json:"serial_number"`
	ExtraDNSLabels              []string               `json:"extra_dns_labels"`
	Ephemeral                   bool                   `json:"ephemeral"`
	LocalFlags                  *localFlagsOutput      `json:"local_flags,omitempty"`
}

type groupDetailOutput struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PeersCount     int    `json:"peers_count"`
	ResourcesCount int    `json:"resources_count"`
	Issued         string `json:"issued,omitempty"`
}

type networkAddressOutput struct {
	NetIP string `json:"net_ip"`
	MAC   string `json:"mac"`
}

// localFlagsOutput leaves out flags the peer did not report, so they are not
// shown as false.
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

// getPeer builds the get_peer tool definition and its handler. The handler
// validates the ID before calling the client and returns raw errors.
func getPeer(c peerAPI) (*mcp.Tool, mcp.ToolHandlerFor[getPeerInput, peerDetailOutput]) {
	tool := &mcp.Tool{
		Name:        "get_peer",
		Description: "Gets all details of one peer (device) in the NetBird account by its ID. Use list_peers to find the ID.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in getPeerInput) (*mcp.CallToolResult, peerDetailOutput, error) {
		if !peerIDPattern.MatchString(in.PeerID) {
			return nil, peerDetailOutput{}, errInvalidPeerID
		}
		peer, err := c.GetPeer(ctx, in.PeerID)
		if err != nil {
			return nil, peerDetailOutput{}, err
		}
		return nil, toPeerDetailOutput(peer), nil
	}
	return tool, handler
}

func toPeerDetailOutput(p netbird.Peer) peerDetailOutput {
	groups := make([]groupDetailOutput, 0, len(p.Groups))
	for _, g := range p.Groups {
		groups = append(groups, groupDetailOutput{
			ID: g.ID, Name: g.Name, PeersCount: g.PeersCount, ResourcesCount: g.ResourcesCount, Issued: g.Issued,
		})
	}
	labels := p.ExtraDNSLabels
	if labels == nil {
		labels = []string{}
	}
	var addrs []networkAddressOutput
	if p.NetworkAddresses != nil {
		addrs = make([]networkAddressOutput, 0, len(p.NetworkAddresses))
		for _, a := range p.NetworkAddresses {
			addrs = append(addrs, networkAddressOutput{NetIP: a.NetIP, MAC: a.MAC})
		}
	}
	var flags *localFlagsOutput
	if f := p.LocalFlags; f != nil {
		o := localFlagsOutput(*f)
		flags = &o
	}
	return peerDetailOutput{
		ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt, IP: p.IP, IPv6: p.IPv6, NetworkAddresses: addrs,
		ConnectionIP: p.ConnectionIP, Connected: p.Connected, LastSeen: p.LastSeen, OS: p.OS,
		KernelVersion: p.KernelVersion, GeonameID: p.GeonameID, Version: p.Version, Groups: groups,
		SSHEnabled: p.SSHEnabled, UserID: p.UserID, Hostname: p.Hostname, UIVersion: p.UIVersion,
		DNSLabel: p.DNSLabel, LoginExpirationEnabled: p.LoginExpirationEnabled, LoginExpired: p.LoginExpired,
		LastLogin: p.LastLogin, InactivityExpirationEnabled: p.InactivityExpirationEnabled,
		ApprovalRequired: p.ApprovalRequired, DisapprovalReason: p.DisapprovalReason,
		CountryCode: p.CountryCode, CityName: p.CityName, SerialNumber: p.SerialNumber,
		ExtraDNSLabels: labels, Ephemeral: p.Ephemeral, LocalFlags: flags,
	}
}
