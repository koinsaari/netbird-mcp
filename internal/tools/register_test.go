package tools

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

const sentinelToken = "nbp_SENTINEL"

type fakePeerAPI struct {
	peers     []netbird.Peer
	listErr   error
	listCalls int
	params    netbird.ListPeersParams

	peer     netbird.Peer
	getErr   error
	getCalls int
	id       string
}

func (f *fakePeerAPI) ListPeers(_ context.Context, p netbird.ListPeersParams) ([]netbird.Peer, error) {
	f.listCalls++
	f.params = p
	return f.peers, f.listErr
}

func (f *fakePeerAPI) GetPeer(_ context.Context, id string) (netbird.Peer, error) {
	f.getCalls++
	f.id = id
	return f.peer, f.getErr
}

func connect(t *testing.T, f *fakePeerAPI) (*mcp.ClientSession, *bytes.Buffer) {
	t.Helper()
	return connectClient(t, f)
}

func connectClient(t *testing.T, c Client) (*mcp.ClientSession, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, &mcp.ServerOptions{Logger: logger})
	Register(s, c, logger)

	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := s.Connect(t.Context(), t1, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() }) // best-effort teardown
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil).Connect(t.Context(), t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() }) // best-effort teardown
	return cs, &logs
}

func call(t *testing.T, cs *mcp.ClientSession, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	return callTool(t, cs, "list_peers", args)
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var text strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	return res, text.String()
}

func TestListPeersTool(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		lister     *fakePeerAPI
		wantErr    string
		wantCalls  int
		wantParams netbird.ListPeersParams
		wantPeers  int
	}{
		{
			name:      "success",
			lister:    &fakePeerAPI{peers: []netbird.Peer{{ID: "a"}, {ID: "b", Groups: []netbird.Group{{ID: "g", Name: "n"}}}}},
			wantCalls: 1, wantPeers: 2,
		},
		{name: "empty", lister: &fakePeerAPI{}, wantCalls: 1},
		{
			name:       "filters canonical ip",
			args:       map[string]any{"name": "lap", "ip": "::ffff:100.64.0.1"},
			lister:     &fakePeerAPI{},
			wantCalls:  1,
			wantParams: netbird.ListPeersParams{Name: "lap", IP: "100.64.0.1"},
		},
		{
			name:      "ip canonicalized",
			args:      map[string]any{"ip": "2001:DB8:0:0::1"},
			lister:    &fakePeerAPI{},
			wantCalls: 1, wantParams: netbird.ListPeersParams{IP: "2001:db8::1"},
		},
		{name: "invalid ip", args: map[string]any{"ip": "nope"}, lister: &fakePeerAPI{}, wantErr: "invalid ip: must be an IP address without zone"},
		{name: "zoned ip", args: map[string]any{"ip": "fe80::1%eth0"}, lister: &fakePeerAPI{}, wantErr: "invalid ip: must be an IP address without zone"},
		{
			name:      "401",
			lister:    &fakePeerAPI{listErr: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantCalls: 1, wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
		{
			name:      "503",
			args:      map[string]any{"name": "distinctname"},
			lister:    &fakePeerAPI{listErr: &netbird.APIError{Status: 503, Message: "down"}},
			wantCalls: 1, wantParams: netbird.ListPeersParams{Name: "distinctname"},
			wantErr: "netbird api: 503: down (hint: NetBird server error, retry later)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connect(t, tt.lister)
			res, text := call(t, cs, tt.args)

			if tt.lister.listCalls != tt.wantCalls {
				t.Errorf("lister calls = %d, want %d", tt.lister.listCalls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && tt.lister.params != tt.wantParams {
				t.Errorf("params = %+v, want %+v", tt.lister.params, tt.wantParams)
			}
			if tt.wantErr != "" {
				if !res.IsError || text != tt.wantErr {
					t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
				}
				if !strings.Contains(logs.String(), "is_error=true") {
					t.Errorf("logs missing is_error=true: %s", logs)
				}
			} else {
				if res.IsError {
					t.Fatalf("unexpected tool error: %s", text)
				}
				sc, ok := res.StructuredContent.(map[string]any)
				if !ok {
					t.Fatalf("structured content = %#v", res.StructuredContent)
				}
				peers, ok := sc["peers"].([]any)
				if !ok || len(peers) != tt.wantPeers {
					t.Errorf("peers = %#v, want %d non-nil", sc["peers"], tt.wantPeers)
				}
				if !strings.Contains(logs.String(), "tool=list_peers") || !strings.Contains(logs.String(), "is_error=false") {
					t.Errorf("logs missing tool fields: %s", logs)
				}
				if tt.wantPeers == 2 {
					groups, ok := peers[0].(map[string]any)["groups"].([]any)
					if !ok || len(groups) != 0 {
						t.Errorf("groups = %#v, want empty non-nil", peers[0].(map[string]any)["groups"])
					}
				}
			}
			if strings.Contains(text, sentinelToken) {
				t.Errorf("sentinel leaked into tool text: %q", text)
			}
			if strings.Contains(logs.String(), "distinctname") {
				t.Errorf("args leaked into logs: %s", logs)
			}
			if tt.name == "401" && !strings.Contains(logs.String(), "level=WARN") {
				t.Errorf("want Warn log, got %s", logs)
			}
		})
	}
}

func TestGetPeerTool(t *testing.T) {
	const id = "cdl0ajd5jtbs73abcdef"
	yes, no := true, false
	full := netbird.Peer{
		ID: id, Name: "laptop", CreatedAt: "2026-01-01T00:00:00Z", IP: "100.64.0.1", IPv6: "fd00::1",
		NetworkAddresses: []netbird.NetworkAddress{{NetIP: "192.168.0.2/24", MAC: "aa:bb"}},
		ConnectionIP:     "1.2.3.4", Connected: true, LastSeen: "2026-01-02T00:00:00Z", OS: "linux",
		KernelVersion: "6.1", GeonameID: 42, Version: "0.1", SSHEnabled: true, UserID: "u1",
		Hostname: "lap", UIVersion: "ui", DNSLabel: "lap", LoginExpirationEnabled: true,
		LastLogin: "2026-01-03T00:00:00Z", InactivityExpirationEnabled: true, ApprovalRequired: true,
		DisapprovalReason: "nope", CountryCode: "FI", CityName: "Helsinki", SerialNumber: "sn",
		ExtraDNSLabels: []string{"x"}, Ephemeral: true,
		Groups:     []netbird.Group{{ID: "g1", Name: "a", PeersCount: 3, ResourcesCount: 2, Issued: "api"}},
		LocalFlags: &netbird.PeerLocalFlags{RosenpassEnabled: &yes, BlockInbound: &no},
	}
	notFound := &netbird.APIError{Status: 404, Message: "peer not found"}
	tests := []struct {
		name      string
		args      map[string]any
		getter    *fakePeerAPI
		wantErr   string
		wantCalls int
	}{
		{name: "success", args: map[string]any{"peer_id": id}, getter: &fakePeerAPI{peer: full}, wantCalls: 1},
		{name: "empty id", args: map[string]any{"peer_id": ""}, getter: &fakePeerAPI{}, wantErr: errInvalidPeerID.Error()},
		{name: "19 chars", args: map[string]any{"peer_id": id[:19]}, getter: &fakePeerAPI{}, wantErr: errInvalidPeerID.Error()},
		{name: "uppercase", args: map[string]any{"peer_id": "CDL0AJD5JTBS73ABCDEF"}, getter: &fakePeerAPI{}, wantErr: errInvalidPeerID.Error()},
		{name: "path traversal", args: map[string]any{"peer_id": "../x"}, getter: &fakePeerAPI{}, wantErr: errInvalidPeerID.Error()},
		{name: "contains w", args: map[string]any{"peer_id": "cdl0ajd5jtbs73abcdew"}, getter: &fakePeerAPI{}, wantErr: errInvalidPeerID.Error()},
		{
			name: "404", args: map[string]any{"peer_id": id}, getter: &fakePeerAPI{getErr: notFound}, wantCalls: 1,
			wantErr: "netbird api: 404: peer not found (hint: not found; check the ID or NETBIRD_API_URL)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectClient(t, tt.getter)
			res, text := callTool(t, cs, "get_peer", tt.args)

			if tt.getter.getCalls != tt.wantCalls {
				t.Errorf("getter calls = %d, want %d", tt.getter.getCalls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && tt.getter.id != id {
				t.Errorf("getter id = %q, want %q", tt.getter.id, id)
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
				for _, k := range []string{
					"id", "name", "created_at", "ip", "ipv6", "network_addresses", "connection_ip", "connected",
					"last_seen", "os", "kernel_version", "geoname_id", "version", "groups", "ssh_enabled", "user_id",
					"hostname", "ui_version", "dns_label", "login_expiration_enabled", "login_expired", "last_login",
					"inactivity_expiration_enabled", "approval_required", "disapproval_reason", "country_code",
					"city_name", "serial_number", "extra_dns_labels", "ephemeral", "local_flags",
				} {
					if _, ok := sc[k]; !ok {
						t.Errorf("structured content missing %q", k)
					}
				}
				flags, _ := sc["local_flags"].(map[string]any)
				if flags["block_inbound"] != false || flags["rosenpass_enabled"] != true || len(flags) != 2 {
					t.Errorf("local_flags = %#v", sc["local_flags"])
				}
				groups, _ := sc["groups"].([]any)
				if len(groups) != 1 || groups[0].(map[string]any)["issued"] != "api" {
					t.Errorf("groups = %#v", sc["groups"])
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestGetPeerToolLeaksNoToken(t *testing.T) {
	t.Parallel()
	g := &fakePeerAPI{getErr: &netbird.APIError{Status: 500, Message: "boom"}}
	cs, logs := connectClient(t, g)
	res, text := callTool(t, cs, "get_peer", map[string]any{"peer_id": "cdl0ajd5jtbs73abcdef"})
	if !res.IsError || strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("IsError=%v text=%q logs=%s", res.IsError, text, logs)
	}
}
