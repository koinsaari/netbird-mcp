package netbird

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListPostureChecksRequest(t *testing.T) {
	t.Parallel()
	var got *http.Request
	c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte("[]"))
	})
	if _, err := c.ListPostureChecks(t.Context()); err != nil {
		t.Fatalf("ListPostureChecks: %v", err)
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.Method)
	}
	if got.URL.Path != "/api/posture-checks" {
		t.Errorf("path = %q", got.URL.Path)
	}
	if got.URL.RawQuery != "" {
		t.Errorf("query = %q, want empty", got.URL.RawQuery)
	}
	if h := got.Header.Get("Authorization"); h != "Token "+sentinelToken {
		t.Errorf("Authorization = %q", h)
	}
	if h := got.Header.Get("Accept"); h != "application/json" {
		t.Errorf("Accept = %q", h)
	}
}

func TestListPostureChecksDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []PostureCheck
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []PostureCheck{}},
		{
			name: "nb version with unknown fields",
			body: `[{"id":"c1","name":"nb","description":"min nb","future":1,
				"checks":{"nb_version_check":{"min_version":"0.30.0","future":1},"future_check":{}}}]`,
			want: []PostureCheck{{
				ID: "c1", Name: "nb", Description: "min nb",
				Checks: PostureChecks{NBVersionCheck: &MinVersionCheck{MinVersion: "0.30.0"}},
			}},
		},
		{
			name: "os version",
			body: `[{"id":"c2","name":"os","checks":{"os_version_check":{
				"android":{"min_version":"13"},"darwin":{"min_version":"14.2.1"},"ios":{"min_version":"17.3.1"},
				"linux":{"min_kernel_version":"5.3.3"},"windows":{"min_kernel_version":"10.0.1234"}}}}]`,
			want: []PostureCheck{{
				ID: "c2", Name: "os",
				Checks: PostureChecks{OSVersionCheck: &OSVersionCheck{
					Android: &MinVersionCheck{MinVersion: "13"},
					Darwin:  &MinVersionCheck{MinVersion: "14.2.1"},
					IOS:     &MinVersionCheck{MinVersion: "17.3.1"},
					Linux:   &MinKernelVersionCheck{MinKernelVersion: "5.3.3"},
					Windows: &MinKernelVersionCheck{MinKernelVersion: "10.0.1234"},
				}},
			}},
		},
		{
			name: "os version partial",
			body: `[{"id":"c3","name":"os","checks":{"os_version_check":{"linux":{"min_kernel_version":"6.6.12"}}}}]`,
			want: []PostureCheck{{
				ID: "c3", Name: "os",
				Checks: PostureChecks{OSVersionCheck: &OSVersionCheck{Linux: &MinKernelVersionCheck{MinKernelVersion: "6.6.12"}}},
			}},
		},
		{
			name: "geo location",
			body: `[{"id":"c4","name":"geo","checks":{"geo_location_check":{"action":"allow",
				"locations":[{"country_code":"DE","city_name":"Berlin"},{"country_code":"FI"}]}}}]`,
			want: []PostureCheck{{
				ID: "c4", Name: "geo",
				Checks: PostureChecks{GeoLocationCheck: &GeoLocationCheck{
					Action:    "allow",
					Locations: []Location{{CountryCode: "DE", CityName: "Berlin"}, {CountryCode: "FI"}},
				}},
			}},
		},
		{
			name: "peer network range",
			body: `[{"id":"c5","name":"net","checks":{"peer_network_range_check":{"action":"deny",
				"ranges":["10.0.0.0/8","2001:db8::/56"]}}}]`,
			want: []PostureCheck{{
				ID: "c5", Name: "net",
				Checks: PostureChecks{PeerNetworkRangeCheck: &PeerNetworkRangeCheck{
					Action: "deny", Ranges: []string{"10.0.0.0/8", "2001:db8::/56"},
				}},
			}},
		},
		{
			name: "process",
			body: `[{"id":"c6","name":"proc","checks":{"process_check":{"processes":[
				{"linux_path":"/usr/bin/nb","mac_path":"/Applications/nb","windows_path":"C:\\nb.exe"},{"linux_path":"/x"}]}}}]`,
			want: []PostureCheck{{
				ID: "c6", Name: "proc",
				Checks: PostureChecks{ProcessCheck: &ProcessCheck{Processes: []Process{
					{LinuxPath: "/usr/bin/nb", MacPath: "/Applications/nb", WindowsPath: `C:\nb.exe`},
					{LinuxPath: "/x"},
				}}},
			}},
		},
		{
			name: "certificate",
			body: `[{"id":"c7","name":"cert","checks":{"certificate_check":{"ca_certificates":[
				"-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"]}}}]`,
			want: []PostureCheck{{
				ID: "c7", Name: "cert",
				Checks: PostureChecks{CertificateCheck: &CertificateCheck{
					CACertificates: []string{"-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"},
				}},
			}},
		},
		{
			name: "no checks",
			body: `[{"id":"c8","name":"none","checks":{}}]`,
			want: []PostureCheck{{ID: "c8", Name: "none"}},
		},
		{name: "order kept", body: `[{"id":"b"},{"id":"a"}]`, want: []PostureCheck{{ID: "b"}, {ID: "a"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := serveBody(t, tt.body).ListPostureChecks(t.Context())
			if err != nil {
				t.Fatalf("ListPostureChecks: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListPostureChecksAPIError(t *testing.T) {
	t.Parallel()
	errs := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{name: "json message", status: 403, body: `{"message":"no permission"}`, wantMessage: "no permission"},
		{name: "token echoed", status: 400, body: `{"message":"bad token nbp_SENTINEL here"}`, wantMessage: "Bad Request"},
	}
	for _, e := range errs {
		t.Run(e.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(e.status)
				_, _ = w.Write([]byte(e.body))
			})
			_, err := c.ListPostureChecks(t.Context())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *APIError", err)
			}
			if apiErr.Status != e.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, e.status)
			}
			if apiErr.Message != e.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, e.wantMessage)
			}
			if !strings.HasPrefix(err.Error(), "list posture checks: ") {
				t.Errorf("err = %q, want prefix", err)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}
