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

type fakeLister struct {
	peers  []netbird.Peer
	err    error
	calls  int
	params netbird.ListPeersParams
}

func (f *fakeLister) ListPeers(_ context.Context, p netbird.ListPeersParams) ([]netbird.Peer, error) {
	f.calls++
	f.params = p
	return f.peers, f.err
}

func connect(t *testing.T, f *fakeLister) (*mcp.ClientSession, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, &mcp.ServerOptions{Logger: logger})
	Register(s, f, logger)

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
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_peers", Arguments: args})
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
		lister     *fakeLister
		wantErr    string
		wantCalls  int
		wantParams netbird.ListPeersParams
		wantPeers  int
	}{
		{
			name:      "success",
			lister:    &fakeLister{peers: []netbird.Peer{{ID: "a"}, {ID: "b", Groups: []netbird.Group{{ID: "g", Name: "n"}}}}},
			wantCalls: 1, wantPeers: 2,
		},
		{name: "empty", lister: &fakeLister{}, wantCalls: 1},
		{
			name:       "filters canonical ip",
			args:       map[string]any{"name": "lap", "ip": "::ffff:100.64.0.1"},
			lister:     &fakeLister{},
			wantCalls:  1,
			wantParams: netbird.ListPeersParams{Name: "lap", IP: "100.64.0.1"},
		},
		{
			name:      "ip canonicalized",
			args:      map[string]any{"ip": "2001:DB8:0:0::1"},
			lister:    &fakeLister{},
			wantCalls: 1, wantParams: netbird.ListPeersParams{IP: "2001:db8::1"},
		},
		{name: "invalid ip", args: map[string]any{"ip": "nope"}, lister: &fakeLister{}, wantErr: "invalid ip: must be an IP address without zone"},
		{name: "zoned ip", args: map[string]any{"ip": "fe80::1%eth0"}, lister: &fakeLister{}, wantErr: "invalid ip: must be an IP address without zone"},
		{
			name:      "401",
			lister:    &fakeLister{err: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantCalls: 1, wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
		{
			name:      "503",
			args:      map[string]any{"name": "distinctname"},
			lister:    &fakeLister{err: &netbird.APIError{Status: 503, Message: "down"}},
			wantCalls: 1, wantParams: netbird.ListPeersParams{Name: "distinctname"},
			wantErr: "netbird api: 503: down (hint: NetBird server error, retry later)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connect(t, tt.lister)
			res, text := call(t, cs, tt.args)

			if tt.lister.calls != tt.wantCalls {
				t.Errorf("lister calls = %d, want %d", tt.lister.calls, tt.wantCalls)
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
