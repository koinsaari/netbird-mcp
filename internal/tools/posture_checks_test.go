package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type fakePostureCheckAPI struct {
	checks []netbird.PostureCheck
	err    error
	calls  atomic.Int32
}

func (f *fakePostureCheckAPI) ListPostureChecks(context.Context) ([]netbird.PostureCheck, error) {
	f.calls.Add(1)
	return f.checks, f.err
}

const pemBody = "-----BEGIN CERTIFICATE-----SECRETPEMBODY-----END CERTIFICATE-----"

func marshalOutput(t *testing.T, checks []netbird.PostureCheck, policies []netbird.Policy) string {
	t.Helper()
	b, err := json.Marshal(toListPostureChecksOutput(checks, policies))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestToListPostureChecksOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		checks   []netbird.PostureCheck
		policies []netbird.Policy
		want     string
	}{
		{name: "nil", want: `{"posture_checks":[]}`},
		{
			name:   "no checks configured",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "empty"}},
			want:   `{"posture_checks":[{"id":"c1","name":"empty","checks":{},"used_by_policies":[]}]}`,
		},
		{
			name: "nb version with description",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "nb", Description: "d",
				Checks: netbird.PostureChecks{NBVersionCheck: &netbird.MinVersionCheck{MinVersion: "0.50.0"}}}},
			want: `{"posture_checks":[{"id":"c1","name":"nb","description":"d","checks":{"nb_version":{"min_version":"0.50.0"}},"used_by_policies":[]}]}`,
		},
		{
			name: "os version full",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "os", Checks: netbird.PostureChecks{OSVersionCheck: &netbird.OSVersionCheck{
				Android: &netbird.MinVersionCheck{MinVersion: "14"},
				Darwin:  &netbird.MinVersionCheck{MinVersion: "14.1"},
				IOS:     &netbird.MinVersionCheck{MinVersion: "17"},
				Linux:   &netbird.MinKernelVersionCheck{MinKernelVersion: "6.1"},
				Windows: &netbird.MinKernelVersionCheck{MinKernelVersion: "10.0.19045"},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"os","checks":{"os_version":{"android":"14","darwin":"14.1","ios":"17","linux_kernel":"6.1","windows_kernel":"10.0.19045"}},"used_by_policies":[]}]}`,
		},
		{
			name: "os version partial",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "os", Checks: netbird.PostureChecks{OSVersionCheck: &netbird.OSVersionCheck{
				Darwin: &netbird.MinVersionCheck{MinVersion: "14"},
				Linux:  &netbird.MinKernelVersionCheck{MinKernelVersion: "6.1"},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"os","checks":{"os_version":{"darwin":"14","linux_kernel":"6.1"}},"used_by_policies":[]}]}`,
		},
		{
			name: "os version empty object",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "os", Checks: netbird.PostureChecks{
				OSVersionCheck: &netbird.OSVersionCheck{}}}},
			want: `{"posture_checks":[{"id":"c1","name":"os","checks":{"os_version":{}},"used_by_policies":[]}]}`,
		},
		{
			name: "geo location",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "geo", Checks: netbird.PostureChecks{GeoLocationCheck: &netbird.GeoLocationCheck{
				Action:    "allow",
				Locations: []netbird.Location{{CountryCode: "FI", CityName: "Helsinki"}, {CountryCode: "SE"}},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"geo","checks":{"geo_location":{"action":"allow","locations":[{"country_code":"FI","city_name":"Helsinki"},{"country_code":"SE"}]}},"used_by_policies":[]}]}`,
		},
		{
			name: "geo location nil locations",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "geo", Checks: netbird.PostureChecks{
				GeoLocationCheck: &netbird.GeoLocationCheck{Action: "deny"}}}},
			want: `{"posture_checks":[{"id":"c1","name":"geo","checks":{"geo_location":{"action":"deny","locations":[]}},"used_by_policies":[]}]}`,
		},
		{
			name: "peer network range",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "net", Checks: netbird.PostureChecks{PeerNetworkRangeCheck: &netbird.PeerNetworkRangeCheck{
				Action: "allow", Ranges: []string{"10.0.0.0/8"},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"net","checks":{"peer_network_range":{"action":"allow","ranges":["10.0.0.0/8"]}},"used_by_policies":[]}]}`,
		},
		{
			name: "peer network range nil ranges",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "net", Checks: netbird.PostureChecks{
				PeerNetworkRangeCheck: &netbird.PeerNetworkRangeCheck{Action: "deny"}}}},
			want: `{"posture_checks":[{"id":"c1","name":"net","checks":{"peer_network_range":{"action":"deny","ranges":[]}},"used_by_policies":[]}]}`,
		},
		{
			name: "process",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "proc", Checks: netbird.PostureChecks{ProcessCheck: &netbird.ProcessCheck{
				Processes: []netbird.Process{{LinuxPath: "/usr/bin/av"}, {MacPath: "/Applications/AV", WindowsPath: `C:\av.exe`}},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"proc","checks":{"process":{"processes":[{"linux_path":"/usr/bin/av"},{"mac_path":"/Applications/AV","windows_path":"C:\\av.exe"}]}},"used_by_policies":[]}]}`,
		},
		{
			name: "process nil processes",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "proc", Checks: netbird.PostureChecks{
				ProcessCheck: &netbird.ProcessCheck{}}}},
			want: `{"posture_checks":[{"id":"c1","name":"proc","checks":{"process":{"processes":[]}},"used_by_policies":[]}]}`,
		},
		{
			name: "certificate count only",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "cert", Checks: netbird.PostureChecks{CertificateCheck: &netbird.CertificateCheck{
				CACertificates: []string{pemBody, pemBody},
			}}}},
			want: `{"posture_checks":[{"id":"c1","name":"cert","checks":{"certificate":{"ca_certificates_count":2}},"used_by_policies":[]}]}`,
		},
		{
			name: "certificate nil",
			checks: []netbird.PostureCheck{{ID: "c1", Name: "cert", Checks: netbird.PostureChecks{
				CertificateCheck: &netbird.CertificateCheck{}}}},
			want: `{"posture_checks":[{"id":"c1","name":"cert","checks":{"certificate":{"ca_certificates_count":0}},"used_by_policies":[]}]}`,
		},
		{
			name: "used by policies match no match multiple disabled and order",
			checks: []netbird.PostureCheck{
				{ID: "c2", Name: "second"},
				{ID: "c1", Name: "first"},
				{ID: "c3", Name: "unused"},
			},
			policies: []netbird.Policy{
				{ID: "p1", Name: "one", Enabled: true, SourcePostureChecks: []string{"c1", "c2"}},
				{ID: "p2", Name: "two", Enabled: false, SourcePostureChecks: []string{"c1"}},
				{ID: "p3", Name: "three", Enabled: true},
				{ID: "p4", Name: "four", Enabled: true, SourcePostureChecks: []string{"other"}},
			},
			want: `{"posture_checks":[` +
				`{"id":"c2","name":"second","checks":{},"used_by_policies":[{"id":"p1","name":"one","enabled":true}]},` +
				`{"id":"c1","name":"first","checks":{},"used_by_policies":[{"id":"p1","name":"one","enabled":true},{"id":"p2","name":"two","enabled":false}]},` +
				`{"id":"c3","name":"unused","checks":{},"used_by_policies":[]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := marshalOutput(t, tt.checks, tt.policies)
			if got != tt.want {
				t.Errorf("output =\n%s\nwant\n%s", got, tt.want)
			}
			if strings.Contains(got, "CERTIFICATE") || strings.Contains(got, "SECRETPEMBODY") {
				t.Errorf("PEM text leaked: %s", got)
			}
		})
	}
}

func TestListPostureChecksTool(t *testing.T) {
	t.Parallel()
	pc := &fakePostureCheckAPI{checks: []netbird.PostureCheck{
		{ID: "c1", Name: "nb", Checks: netbird.PostureChecks{
			NBVersionCheck:   &netbird.MinVersionCheck{MinVersion: "0.50.0"},
			CertificateCheck: &netbird.CertificateCheck{CACertificates: []string{pemBody}},
		}},
		{ID: "c2", Name: "bare"},
	}}
	pol := &fakePolicyAPI{policies: []netbird.Policy{{ID: "p1", Name: "one", Enabled: true, SourcePostureChecks: []string{"c1"}}}}
	cs, logs := connectFake(t, fakeClient{postureCheckAPI: pc, policyAPI: pol})
	res, text := callTool(t, cs, "list_posture_checks", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var gotV, wantV any
	const want = `{"posture_checks":[` +
		`{"id":"c1","name":"nb","checks":{"nb_version":{"min_version":"0.50.0"},"certificate":{"ca_certificates_count":1}},"used_by_policies":[{"id":"p1","name":"one","enabled":true}]},` +
		`{"id":"c2","name":"bare","checks":{},"used_by_policies":[]}]}`
	if err := json.Unmarshal(raw, &gotV); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotV, wantV) {
		t.Errorf("structured =\n%s\nwant\n%s", raw, want)
	}
	if pc.calls.Load() != 1 || pol.calls != 1 {
		t.Errorf("calls = %d/%d, want 1/1", pc.calls.Load(), pol.calls)
	}
	if strings.Contains(text, "SECRETPEMBODY") || strings.Contains(string(raw), "SECRETPEMBODY") {
		t.Errorf("PEM leaked: text=%q", text)
	}
	if !strings.Contains(logs.String(), "tool=list_posture_checks") || !strings.Contains(logs.String(), "is_error=false") {
		t.Errorf("logs missing tool fields: %s", logs)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestListPostureChecksToolEmpty(t *testing.T) {
	t.Parallel()
	cs, _ := connectFake(t, fakeClient{postureCheckAPI: &fakePostureCheckAPI{}, policyAPI: &fakePolicyAPI{}})
	res, text := callTool(t, cs, "list_posture_checks", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	sc, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content = %#v", res.StructuredContent)
	}
	checks, ok := sc["posture_checks"].([]any)
	if !ok || len(checks) != 0 {
		t.Errorf("posture_checks = %#v, want empty non-nil", sc["posture_checks"])
	}
}

func TestListPostureChecksToolErrors(t *testing.T) {
	t.Parallel()
	unauthorized := &netbird.APIError{Status: 401, Message: "bad token"}
	const want401 = "netbird api: 401: bad token (hint: token invalid or expired)"
	tests := []struct {
		name    string
		checks  *fakePostureCheckAPI
		pol     *fakePolicyAPI
		wantErr string
	}{
		{
			name:    "posture checks fail with 401",
			checks:  &fakePostureCheckAPI{err: unauthorized},
			pol:     &fakePolicyAPI{},
			wantErr: want401,
		},
		{
			name:    "policies fail with 401",
			checks:  &fakePostureCheckAPI{checks: []netbird.PostureCheck{{ID: "c1"}}},
			pol:     &fakePolicyAPI{err: unauthorized},
			wantErr: want401,
		},
		{
			name:    "policies fail with 503",
			checks:  &fakePostureCheckAPI{},
			pol:     &fakePolicyAPI{err: &netbird.APIError{Status: 503, Message: "down"}},
			wantErr: "netbird api: 503: down (hint: NetBird server error, retry later)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{postureCheckAPI: tt.checks, policyAPI: tt.pol})
			res, text := callTool(t, cs, "list_posture_checks", nil)
			if !res.IsError || text != tt.wantErr {
				t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
			}
			if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "is_error=true") {
				t.Errorf("want Warn log, got %s", logs)
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListPostureChecksToolDefinition(t *testing.T) {
	t.Parallel()
	tool, _ := listPostureChecks(struct {
		postureCheckAPI
		policyAPI
	}{&fakePostureCheckAPI{}, &fakePolicyAPI{}})
	if tool.Name != "list_posture_checks" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Lists all NetBird posture checks with their configured checks (NetBird and OS minimum versions, geo location, peer network ranges, processes, certificate count) and the policies that require each check. A posture check only takes effect through the policies listed."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
	cs, _ := connectFake(t, fakeClient{postureCheckAPI: &fakePostureCheckAPI{}, policyAPI: &fakePolicyAPI{}})
	list, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, lt := range list.Tools {
		if lt.Name != "list_posture_checks" {
			continue
		}
		in, err := json.Marshal(lt.InputSchema)
		if err != nil {
			t.Fatalf("marshal input schema: %v", err)
		}
		if strings.Contains(string(in), `"properties":{"`) {
			t.Errorf("input schema has properties: %s", in)
		}
		return
	}
	t.Error("list_posture_checks not registered")
}
