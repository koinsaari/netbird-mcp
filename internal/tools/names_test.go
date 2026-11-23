package tools

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestResolveNames(t *testing.T) {
	groupsErr := &netbird.APIError{Status: 500, Message: "groups boom"}
	peersErr := &netbird.APIError{Status: 502, Message: "peers boom"}
	coreErr := errors.New("core boom")
	groupList := []netbird.Group{{ID: "g1", Name: "devs"}}
	peerList := []netbird.Peer{{ID: "p1", Name: "laptop"}}

	tests := []struct {
		name       string
		groups     *fakeGroupAPI
		peers      *fakePeerAPI
		nilPeers   bool
		coreErr    error
		wantErr    error
		wantGroups map[string]string
		wantPeers  map[string]string
		wantNames  string
	}{
		{
			name:       "success",
			groups:     &fakeGroupAPI{groups: groupList},
			peers:      &fakePeerAPI{peers: peerList},
			wantGroups: map[string]string{"g1": "devs"},
			wantPeers:  map[string]string{"p1": "laptop"},
		},
		{
			name:       "groups failure",
			groups:     &fakeGroupAPI{groups: groupList, err: groupsErr},
			peers:      &fakePeerAPI{peers: peerList},
			wantGroups: map[string]string{},
			wantPeers:  map[string]string{"p1": "laptop"},
			wantNames:  toolMessage(groupsErr),
		},
		{
			name:       "peers failure",
			groups:     &fakeGroupAPI{groups: groupList},
			peers:      &fakePeerAPI{peers: peerList, listErr: peersErr},
			wantGroups: map[string]string{"g1": "devs"},
			wantPeers:  map[string]string{},
			wantNames:  toolMessage(peersErr),
		},
		{
			name:       "both fail reports groups error",
			groups:     &fakeGroupAPI{err: groupsErr},
			peers:      &fakePeerAPI{listErr: peersErr},
			wantGroups: map[string]string{},
			wantPeers:  map[string]string{},
			wantNames:  toolMessage(groupsErr),
		},
		{
			name:    "core failure",
			groups:  &fakeGroupAPI{groups: groupList},
			peers:   &fakePeerAPI{peers: peerList},
			coreErr: coreErr,
			wantErr: coreErr,
		},
		{
			name:       "nil peer api",
			groups:     &fakeGroupAPI{groups: groupList},
			nilPeers:   true,
			wantGroups: map[string]string{"g1": "devs"},
			wantPeers:  map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p peerAPI
			if !tt.nilPeers {
				p = tt.peers
			}
			coreCalls := 0
			got, err := resolveNames(t.Context(), tt.groups, p, func(context.Context) error {
				coreCalls++
				return tt.coreErr
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if coreCalls != 1 {
				t.Errorf("core calls = %d, want 1", coreCalls)
			}
			if !tt.nilPeers && tt.peers.listCalls != 1 {
				t.Errorf("ListPeers calls = %d, want 1", tt.peers.listCalls)
			}
			if tt.wantErr != nil {
				if got.groups != nil || got.peers != nil || got.err != "" {
					t.Errorf("got %+v, want zero refNames", got)
				}
				return
			}
			if got.groups == nil || got.peers == nil {
				t.Fatalf("maps must not be nil: %+v", got)
			}
			if !maps.Equal(got.groups, tt.wantGroups) {
				t.Errorf("groups = %v, want %v", got.groups, tt.wantGroups)
			}
			if !maps.Equal(got.peers, tt.wantPeers) {
				t.Errorf("peers = %v, want %v", got.peers, tt.wantPeers)
			}
			if got.err != tt.wantNames {
				t.Errorf("err = %q, want %q", got.err, tt.wantNames)
			}
		})
	}
}
