package tools

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestToPeerOutput(t *testing.T) {
	full := netbird.Peer{
		ID: "p1", Name: "n", Hostname: "h", DNSLabel: "d", IP: "100.64.0.1", Connected: true,
		LastSeen: "2026-01-01T00:00:00Z", OS: "linux", Version: "0.1", UserID: "u",
		LoginExpired: true, ApprovalRequired: true,
		Groups: []netbird.Group{{ID: "g1", Name: "a"}, {ID: "g2", Name: "b"}},
	}
	tests := []struct {
		name string
		in   netbird.Peer
		want peerOutput
	}{
		{"nil groups", netbird.Peer{ID: "p"}, peerOutput{ID: "p", Groups: []groupOutput{}}},
		{"full", full, peerOutput{
			ID: "p1", Name: "n", Hostname: "h", DNSLabel: "d", IP: "100.64.0.1", Connected: true,
			LastSeen: "2026-01-01T00:00:00Z", OS: "linux", Version: "0.1", UserID: "u",
			LoginExpired: true, ApprovalRequired: true,
			Groups: []groupOutput{{ID: "g1", Name: "a"}, {ID: "g2", Name: "b"}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := toPeerOutput(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToListPeersOutput(t *testing.T) {
	tests := []struct {
		name    string
		in      []netbird.Peer
		wantIDs []string
	}{
		{"nil", nil, []string{}},
		{"empty", []netbird.Peer{}, []string{}},
		{"order preserved", []netbird.Peer{{ID: "b"}, {ID: "a"}}, []string{"b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toListPeersOutput(tt.in)
			if got.Peers == nil {
				t.Fatal("Peers is nil")
			}
			ids := make([]string, 0, len(got.Peers))
			for _, p := range got.Peers {
				ids = append(ids, p.ID)
			}
			if !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tt.wantIDs)
			}
		})
	}
}

func TestToPeerDetailOutput(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name     string
		in       netbird.Peer
		wantKeys []string
		absent   []string
		check    func(t *testing.T, got map[string]any)
	}{
		{
			name:     "nil slices and absent optionals",
			in:       netbird.Peer{ID: "p"},
			wantKeys: []string{"id", "groups", "extra_dns_labels", "geoname_id", "connected"},
			absent:   []string{"ipv6", "disapproval_reason", "network_addresses", "local_flags"},
			check: func(t *testing.T, got map[string]any) {
				for _, k := range []string{"groups", "extra_dns_labels"} {
					if v, ok := got[k].([]any); !ok || len(v) != 0 {
						t.Errorf("%s = %#v, want []", k, got[k])
					}
				}
			},
		},
		{
			name: "present optionals",
			in: netbird.Peer{
				ID: "p", IPv6: "fd00::1", DisapprovalReason: "r",
				NetworkAddresses: []netbird.NetworkAddress{{NetIP: "1.2.3.4/24", MAC: "m"}},
				Groups:           []netbird.Group{{ID: "g", Name: "n", PeersCount: 1, ResourcesCount: 2, Issued: "jwt"}},
			},
			wantKeys: []string{"ipv6", "disapproval_reason", "network_addresses"},
			absent:   []string{"local_flags"},
			check: func(t *testing.T, got map[string]any) {
				g := got["groups"].([]any)[0].(map[string]any)
				if g["peers_count"] != float64(1) || g["resources_count"] != float64(2) || g["issued"] != "jwt" {
					t.Errorf("group = %#v", g)
				}
			},
		},
		{
			name:   "group without issued",
			in:     netbird.Peer{Groups: []netbird.Group{{ID: "g"}}},
			absent: nil,
			check: func(t *testing.T, got map[string]any) {
				g := got["groups"].([]any)[0].(map[string]any)
				if _, ok := g["issued"]; ok {
					t.Errorf("issued present: %#v", g)
				}
				if _, ok := g["peers_count"]; !ok {
					t.Errorf("peers_count missing: %#v", g)
				}
			},
		},
		{
			name:     "flag false versus nil",
			in:       netbird.Peer{LocalFlags: &netbird.PeerLocalFlags{BlockInbound: &no, DisableDNS: &yes}},
			wantKeys: []string{"local_flags"},
			check: func(t *testing.T, got map[string]any) {
				f := got["local_flags"].(map[string]any)
				if v, ok := f["block_inbound"]; !ok || v != false {
					t.Errorf("block_inbound = %#v, want false", v)
				}
				if f["disable_dns"] != true {
					t.Errorf("disable_dns = %#v", f["disable_dns"])
				}
				if _, ok := f["rosenpass_enabled"]; ok || len(f) != 2 {
					t.Errorf("local_flags = %#v, want only the 2 reported", f)
				}
			},
		},
		{
			name:     "empty local flags object",
			in:       netbird.Peer{LocalFlags: &netbird.PeerLocalFlags{}},
			wantKeys: []string{"local_flags"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(toPeerDetailOutput(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			for _, k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Errorf("key %q missing in %s", k, b)
				}
			}
			for _, k := range tt.absent {
				if _, ok := got[k]; ok {
					t.Errorf("key %q present in %s", k, b)
				}
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
