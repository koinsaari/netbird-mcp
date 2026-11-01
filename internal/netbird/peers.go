package netbird

import (
	"context"
	"fmt"
	"net/url"
)

// Peer is a NetBird peer, decoded from the API Peer schema. Optional string
// fields are empty when the API omits them.
type Peer struct {
	ID                          string           `json:"id"`
	Name                        string           `json:"name"`
	CreatedAt                   string           `json:"created_at"`
	IP                          string           `json:"ip"`
	IPv6                        string           `json:"ipv6"`
	NetworkAddresses            []NetworkAddress `json:"network_addresses"`
	ConnectionIP                string           `json:"connection_ip"`
	Connected                   bool             `json:"connected"`
	LastSeen                    string           `json:"last_seen"`
	OS                          string           `json:"os"`
	KernelVersion               string           `json:"kernel_version"`
	GeonameID                   int              `json:"geoname_id"`
	Version                     string           `json:"version"`
	Groups                      []Group          `json:"groups"`
	SSHEnabled                  bool             `json:"ssh_enabled"`
	UserID                      string           `json:"user_id"`
	Hostname                    string           `json:"hostname"`
	UIVersion                   string           `json:"ui_version"`
	DNSLabel                    string           `json:"dns_label"`
	LoginExpirationEnabled      bool             `json:"login_expiration_enabled"`
	LoginExpired                bool             `json:"login_expired"`
	LastLogin                   string           `json:"last_login"`
	InactivityExpirationEnabled bool             `json:"inactivity_expiration_enabled"`
	ApprovalRequired            bool             `json:"approval_required"`
	DisapprovalReason           string           `json:"disapproval_reason"`
	CountryCode                 string           `json:"country_code"`
	CityName                    string           `json:"city_name"`
	SerialNumber                string           `json:"serial_number"`
	ExtraDNSLabels              []string         `json:"extra_dns_labels"`
	Ephemeral                   bool             `json:"ephemeral"`
	LocalFlags                  *PeerLocalFlags  `json:"local_flags"`
}

// NetworkAddress is a network interface reported by a peer.
type NetworkAddress struct {
	NetIP string `json:"net_ip"`
	MAC   string `json:"mac"`
}

// PeerLocalFlags holds the client settings a peer reports. A nil field means
// the peer did not report it, which differs from false.
type PeerLocalFlags struct {
	RosenpassEnabled      *bool `json:"rosenpass_enabled"`
	RosenpassPermissive   *bool `json:"rosenpass_permissive"`
	ServerSSHAllowed      *bool `json:"server_ssh_allowed"`
	RemoteJobsAllowed     *bool `json:"remote_jobs_allowed"`
	DisableClientRoutes   *bool `json:"disable_client_routes"`
	DisableServerRoutes   *bool `json:"disable_server_routes"`
	DisableDNS            *bool `json:"disable_dns"`
	DisableFirewall       *bool `json:"disable_firewall"`
	BlockLANAccess        *bool `json:"block_lan_access"`
	BlockInbound          *bool `json:"block_inbound"`
	LazyConnectionEnabled *bool `json:"lazy_connection_enabled"`
}

// Group is the minimal group reference embedded in other resources.
type Group struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PeersCount     int    `json:"peers_count"`
	ResourcesCount int    `json:"resources_count"`
	Issued         string `json:"issued"`
}

// ListPeersParams filters ListPeers. Empty fields are not sent.
type ListPeersParams struct {
	Name string
	IP   string
}

// ListPeers returns all peers matching params.
func (c *Client) ListPeers(ctx context.Context, params ListPeersParams) ([]Peer, error) {
	query := url.Values{}
	if params.Name != "" {
		query.Set("name", params.Name)
	}
	if params.IP != "" {
		query.Set("ip", params.IP)
	}

	var peers []Peer
	if err := c.get(ctx, &peers, query, "peers"); err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	return peers, nil
}

// GetPeer returns the peer with the given ID. The caller validates id.
func (c *Client) GetPeer(ctx context.Context, id string) (Peer, error) {
	var peer Peer
	if err := c.get(ctx, &peer, nil, "peers", id); err != nil {
		return Peer{}, fmt.Errorf("get peer: %w", err)
	}
	return peer, nil
}
