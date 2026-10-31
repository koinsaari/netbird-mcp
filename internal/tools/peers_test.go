package tools

import (
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
