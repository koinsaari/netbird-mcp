package netbird

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

const sentinelToken = "nbp_SENTINEL"

// newTestClient starts a server running h and returns a client pointed at
// srv.URL+basePath.
func newTestClient(t *testing.T, basePath string, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewTestServer(t, h)
	return New(srv.URL+basePath, sentinelToken, srv.Client())
}

func TestListPeersRequest(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		params    ListPeersParams
		wantPath  string
		wantQuery string
	}{
		{name: "no params", basePath: "/api", wantPath: "/api/peers", wantQuery: ""},
		{name: "trailing slash base", basePath: "/prefix/api/", wantPath: "/prefix/api/peers", wantQuery: ""},
		{name: "name", basePath: "/api", params: ListPeersParams{Name: "my laptop"}, wantPath: "/api/peers", wantQuery: "name=my+laptop"},
		{name: "ip", basePath: "/api", params: ListPeersParams{IP: "100.64.0.1"}, wantPath: "/api/peers", wantQuery: "ip=100.64.0.1"},
		{name: "both", basePath: "/api", params: ListPeersParams{Name: "a", IP: "100.64.0.1"}, wantPath: "/api/peers", wantQuery: "ip=100.64.0.1&name=a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got *http.Request
			c := newTestClient(t, tt.basePath, func(w http.ResponseWriter, r *http.Request) {
				got = r
				_, _ = w.Write([]byte("[]"))
			})
			if _, err := c.ListPeers(t.Context(), tt.params); err != nil {
				t.Fatalf("ListPeers: %v", err)
			}
			if got.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", got.Method)
			}
			if got.URL.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", got.URL.Path, tt.wantPath)
			}
			if got.URL.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.RawQuery, tt.wantQuery)
			}
			if h := got.Header.Get("Authorization"); h != "Token "+sentinelToken {
				t.Errorf("Authorization = %q", h)
			}
			if h := got.Header.Get("Accept"); h != "application/json" {
				t.Errorf("Accept = %q", h)
			}
		})
	}
}

func TestListPeersDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Peer
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []Peer{}},
		{
			name: "peer with unknown fields",
			body: `[{"id":"chacbco6lnnbn6cg5s90","name":"laptop","hostname":"host1","dns_label":"laptop.netbird.cloud",
				"ip":"100.64.0.1","connected":true,"last_seen":"2026-10-31T10:00:00Z","os":"Darwin","version":"0.60.0",
				"groups":[{"id":"g1","name":"All","peers_count":3,"future_group_field":true}],"user_id":"u1","login_expired":false,
				"approval_required":true,"kernel_version":"24.0","geoname_id":1,"future_field":{"x":1}}]`,
			want: []Peer{{
				ID: "chacbco6lnnbn6cg5s90", Name: "laptop", Hostname: "host1", DNSLabel: "laptop.netbird.cloud",
				IP: "100.64.0.1", Connected: true, LastSeen: "2026-10-31T10:00:00Z", OS: "Darwin", Version: "0.60.0",
				Groups: []Group{{ID: "g1", Name: "All", PeersCount: 3}}, UserID: "u1", ApprovalRequired: true,
				KernelVersion: "24.0", GeonameID: 1,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.ListPeers(t.Context(), ListPeersParams{})
			if err != nil {
				t.Fatalf("ListPeers: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListPeersAPIError(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		wantMessage string
	}{
		{name: "json message", status: 403, contentType: "application/json", body: `{"message":"no permission","code":403}`, wantMessage: "no permission"},
		{name: "code ignored", status: 422, contentType: "application/json", body: `{"message":"bad ip","code":500}`, wantMessage: "bad ip"},
		{name: "html body", status: 502, contentType: "text/html", body: "<html>bad gateway</html>", wantMessage: "Bad Gateway"},
		{name: "empty body", status: 500, body: "", wantMessage: "Internal Server Error"},
		{name: "empty message", status: 401, contentType: "application/json", body: `{"message":""}`, wantMessage: "Unauthorized"},
		{name: "token echoed", status: 400, contentType: "application/json", body: `{"message":"bad token nbp_SENTINEL here"}`, wantMessage: "Bad Request"},
		{name: "control chars stripped", status: 400, contentType: "application/json", body: `{"message":"a\u0000b\nc"}`, wantMessage: "ab c"},
		{name: "whitespace collapsed", status: 400, contentType: "application/json", body: `{"message":"invalid ip\n\texpected  IPv4"}`, wantMessage: "invalid ip expected IPv4"},
		{name: "format chars dropped", status: 400, contentType: "application/json", body: `{"message":"a\u202eb\u200bc"}`, wantMessage: "abc"},
		{name: "token split by control char", status: 400, contentType: "application/json", body: `{"message":"bad nbp_\u0000SENTINEL"}`, wantMessage: "Bad Request"},
		{name: "token split by tab", status: 400, contentType: "application/json", body: `{"message":"bad nbp_SEN\tTINEL"}`, wantMessage: "Bad Request"},
		{name: "token split by space", status: 400, contentType: "application/json", body: `{"message":"bad nbp_SEN TINEL"}`, wantMessage: "Bad Request"},
		{name: "long message capped", status: 400, contentType: "application/json", body: `{"message":"` + strings.Repeat("x", 2000) + `"}`, wantMessage: strings.Repeat("x", maxMessageLen)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := c.ListPeers(t.Context(), ListPeersParams{})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *APIError", err)
			}
			if apiErr.Status != tt.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, tt.status)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}

func TestGetPeerRequest(t *testing.T) {
	t.Parallel()
	var got *http.Request
	c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte("{}"))
	})
	if _, err := c.GetPeer(t.Context(), "chacbco6lnnbn6cg5s90"); err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.Method)
	}
	if got.URL.Path != "/api/peers/chacbco6lnnbn6cg5s90" {
		t.Errorf("path = %q", got.URL.Path)
	}
	if h := got.Header.Get("Authorization"); h != "Token "+sentinelToken {
		t.Errorf("Authorization = %q", h)
	}
	if h := got.Header.Get("Accept"); h != "application/json" {
		t.Errorf("Accept = %q", h)
	}
}

func TestGetPeerDecodeFull(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chacbco6lnnbn6cg5s90","name":"laptop","created_at":"2026-01-01T00:00:00Z",
			"ip":"100.64.0.1","ipv6":"fd00::1","network_addresses":[{"net_ip":"192.168.0.11/24","mac":"00:93:37:bd:83:0f"}],
			"connection_ip":"35.64.0.1","connected":true,"last_seen":"2026-10-31T10:00:00Z","os":"Darwin 13.2.1",
			"kernel_version":"23.2.0","geoname_id":2643743,"version":"0.60.0",
			"groups":[{"id":"g1","name":"All","peers_count":3,"resources_count":5,"issued":"api"}],
			"ssh_enabled":true,"user_id":"u1","hostname":"host1","ui_version":"0.61.0","dns_label":"laptop.netbird.cloud",
			"login_expiration_enabled":true,"login_expired":false,"last_login":"2026-01-02T00:00:00Z",
			"inactivity_expiration_enabled":true,"approval_required":true,"disapproval_reason":"unknown device",
			"country_code":"DE","city_name":"Berlin","serial_number":"C02XJ0J0JGH7","extra_dns_labels":["alias"],
			"ephemeral":true,"local_flags":{"rosenpass_enabled":true,"rosenpass_permissive":true,"server_ssh_allowed":true,
			"remote_jobs_allowed":true,"disable_client_routes":true,"disable_server_routes":true,"disable_dns":true,
			"disable_firewall":true,"block_lan_access":true,"block_inbound":true,"lazy_connection_enabled":true}}`))
	})
	got, err := c.GetPeer(t.Context(), "chacbco6lnnbn6cg5s90")
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	tr := true
	want := Peer{
		ID: "chacbco6lnnbn6cg5s90", Name: "laptop", CreatedAt: "2026-01-01T00:00:00Z",
		IP: "100.64.0.1", IPv6: "fd00::1",
		NetworkAddresses: []NetworkAddress{{NetIP: "192.168.0.11/24", MAC: "00:93:37:bd:83:0f"}},
		ConnectionIP:     "35.64.0.1", Connected: true, LastSeen: "2026-10-31T10:00:00Z", OS: "Darwin 13.2.1",
		KernelVersion: "23.2.0", GeonameID: 2643743, Version: "0.60.0",
		Groups:     []Group{{ID: "g1", Name: "All", PeersCount: 3, ResourcesCount: 5, Issued: "api"}},
		SSHEnabled: true, UserID: "u1", Hostname: "host1", UIVersion: "0.61.0", DNSLabel: "laptop.netbird.cloud",
		LoginExpirationEnabled: true, LastLogin: "2026-01-02T00:00:00Z", InactivityExpirationEnabled: true,
		ApprovalRequired: true, DisapprovalReason: "unknown device", CountryCode: "DE", CityName: "Berlin",
		SerialNumber: "C02XJ0J0JGH7", ExtraDNSLabels: []string{"alias"}, Ephemeral: true,
		LocalFlags: &PeerLocalFlags{
			RosenpassEnabled: &tr, RosenpassPermissive: &tr, ServerSSHAllowed: &tr, RemoteJobsAllowed: &tr,
			DisableClientRoutes: &tr, DisableServerRoutes: &tr, DisableDNS: &tr, DisableFirewall: &tr,
			BlockLANAccess: &tr, BlockInbound: &tr, LazyConnectionEnabled: &tr,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetPeerDecodeOptionalAbsent(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"p1","ip":"100.64.0.1","groups":[],"extra_dns_labels":[]}`))
	})
	got, err := c.GetPeer(t.Context(), "p1")
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got.LocalFlags != nil {
		t.Errorf("LocalFlags = %+v, want nil", got.LocalFlags)
	}
	if got.NetworkAddresses != nil {
		t.Errorf("NetworkAddresses = %+v, want nil", got.NetworkAddresses)
	}
	if got.IPv6 != "" || got.DisapprovalReason != "" {
		t.Errorf("IPv6 = %q, DisapprovalReason = %q, want empty", got.IPv6, got.DisapprovalReason)
	}
}

func TestGetPeerLocalFlagFalseVsAbsent(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"p1","local_flags":{"block_inbound":false}}`))
	})
	got, err := c.GetPeer(t.Context(), "p1")
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got.LocalFlags == nil {
		t.Fatal("LocalFlags = nil, want non-nil")
	}
	if got.LocalFlags.BlockInbound == nil || *got.LocalFlags.BlockInbound {
		t.Errorf("BlockInbound = %v, want pointer to false", got.LocalFlags.BlockInbound)
	}
	if got.LocalFlags.DisableDNS != nil {
		t.Errorf("DisableDNS = %v, want nil", got.LocalFlags.DisableDNS)
	}
}

func TestGetPeerNotFound(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"peer not found nbp_SENTINEL","code":404}`))
	})
	_, err := c.GetPeer(t.Context(), "chacbco6lnnbn6cg5s90")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
	if strings.Contains(err.Error(), sentinelToken) {
		t.Errorf("err leaks token: %q", err)
	}
	if !strings.HasPrefix(err.Error(), "get peer: ") {
		t.Errorf("err = %q, want get peer prefix", err)
	}
}

func TestAPIErrorString(t *testing.T) {
	t.Parallel()
	err := &APIError{Status: 403, Message: "no permission"}
	if got, want := err.Error(), "netbird api: 403: no permission"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestListPeersFailures(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		cancel  bool
		maxBody int64
		check   func(t *testing.T, err error)
	}{
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`[{"id": ` + sentinelToken + `}]`))
			},
			check: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "decode response: response is not JSON") {
					t.Errorf("err = %v, want invalid JSON", err)
				}
			},
		},
		{
			name: "body cut off mid-stream",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte(`[{"id":"a"`))
			},
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("err = %v, want io.ErrUnexpectedEOF", err)
				}
			},
		},
		{
			name: "wrong json type",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"message":"` + sentinelToken + `"}`))
			},
			check: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "cannot decode JSON") {
					t.Errorf("err = %v, want cannot decode JSON", err)
				}
			},
		},
		{
			name:    "body too large",
			maxBody: 10,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`[{"id":"aaaaaaaaaaaaaaaaaaaa"}]`))
			},
			check: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "too large") {
					t.Errorf("err = %v, want too large", err)
				}
			},
		},
		{
			name:    "canceled context",
			handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("[]")) },
			cancel:  true,
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, context.Canceled) {
					t.Errorf("err = %v, want context.Canceled", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", tt.handler)
			if tt.maxBody > 0 {
				c.maxBodySize = tt.maxBody
			}
			ctx := t.Context()
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, err := c.ListPeers(ctx, ListPeersParams{})
			if err == nil {
				t.Fatal("err = nil, want error")
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
			if tt.check != nil {
				tt.check(t, err)
			}
		})
	}
}

func TestGetPathSegments(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		wantPath string
		wantErr  bool
	}{
		{name: "plain", segments: []string{"peers", "chacbco6lnnbn6cg5s90"}, wantPath: "/api/peers/chacbco6lnnbn6cg5s90"},
		{name: "slash escaped", segments: []string{"peers", "../users"}, wantPath: "/api/peers/..%2Fusers"},
		{name: "dot dot rejected", segments: []string{"peers", ".."}, wantErr: true},
		{name: "dot rejected", segments: []string{"peers", "."}, wantErr: true},
		{name: "empty rejected", segments: []string{"peers", ""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotPath string
			called := false
			c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotPath = r.URL.EscapedPath()
				_, _ = w.Write([]byte("{}"))
			})
			var out struct{}
			err := c.get(t.Context(), &out, nil, tt.segments...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("err = nil, want error")
				}
				if called {
					t.Error("request sent for invalid segment")
				}
				return
			}
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestListPeersTimeoutMidBody(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"a"`))
		w.(http.Flusher).Flush()
		<-r.Context().Done() // hold the body open until the client gives up
	})
	c.hc = &http.Client{Transport: c.hc.Transport, Timeout: 50 * time.Millisecond}

	_, err := c.ListPeers(t.Context(), ListPeersParams{})
	netErr, ok := errors.AsType[net.Error](err)
	if !errors.Is(err, context.DeadlineExceeded) && (!ok || !netErr.Timeout()) {
		t.Fatalf("err = %v, want a timeout", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestListPeersTransportErrorOmitsQuery(t *testing.T) {
	tests := []struct {
		name  string
		cause error
		check func(t *testing.T, err error)
	}{
		{"timeout", context.DeadlineExceeded, func(t *testing.T, err error) {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("errors.Is(DeadlineExceeded) = false for %v", err)
			}
		}},
		{"refused", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, func(t *testing.T, err error) {
			if _, ok := errors.AsType[net.Error](err); !ok {
				t.Errorf("errors.AsType[net.Error] failed for %v", err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, tt.cause
			})}
			c := New("http://example.invalid/api", sentinelToken, hc)
			_, err := c.ListPeers(t.Context(), ListPeersParams{Name: "secretname"})
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "secretname") || strings.Contains(err.Error(), "?") {
				t.Errorf("error leaks query: %q", err)
			}
			tt.check(t, err)
		})
	}
}
