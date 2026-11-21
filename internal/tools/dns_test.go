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

type fakeDNSAPI struct {
	nsGroups    []netbird.NameserverGroup
	nsErr       error
	settings    netbird.DNSSettings
	settingsErr error
	zones       []netbird.DNSZone
	zonesErr    error
	calls       atomic.Int32
}

func (f *fakeDNSAPI) ListNameserverGroups(context.Context) ([]netbird.NameserverGroup, error) {
	f.calls.Add(1)
	return f.nsGroups, f.nsErr
}

func (f *fakeDNSAPI) GetDNSSettings(context.Context) (netbird.DNSSettings, error) {
	f.calls.Add(1)
	return f.settings, f.settingsErr
}

func (f *fakeDNSAPI) ListDNSZones(context.Context) ([]netbird.DNSZone, error) {
	f.calls.Add(1)
	return f.zones, f.zonesErr
}

func marshalDNSOutput(t *testing.T, ns []netbird.NameserverGroup, s netbird.DNSSettings, z []netbird.DNSZone, g []netbird.Group) string {
	t.Helper()
	b, err := json.Marshal(toListDNSOutput(ns, s, z, g))
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	return string(b)
}

func TestToListDNSOutput(t *testing.T) {
	t.Parallel()
	groups := []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "ops"}}
	tests := []struct {
		name     string
		ns       []netbird.NameserverGroup
		settings netbird.DNSSettings
		zones    []netbird.DNSZone
		groups   []netbird.Group
		want     string
	}{
		{name: "nil", want: `{"nameserver_groups":[],"zones":[],"disabled_management_groups":[]}`},
		{
			name: "nameserver group all fields",
			ns: []netbird.NameserverGroup{{
				ID: "n1", Name: "cf", Description: "cloudflare", Enabled: true, Primary: true,
				Domains: []string{"example.com"}, SearchDomainsEnabled: true,
				Groups:      []string{"g2", "g1"},
				Nameservers: []netbird.Nameserver{{IP: "1.1.1.1", NSType: "udp", Port: 53}, {IP: "1.0.0.1", NSType: "udp", Port: 5353}},
			}},
			groups: groups,
			want: `{"nameserver_groups":[{"id":"n1","name":"cf","description":"cloudflare","enabled":true,"primary":true,` +
				`"domains":["example.com"],"search_domains_enabled":true,"groups":[{"id":"g2","name":"ops"},{"id":"g1","name":"devs"}],` +
				`"nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53},{"ip":"1.0.0.1","ns_type":"udp","port":5353}]}],` +
				`"zones":[],"disabled_management_groups":[]}`,
		},
		{
			name: "nameserver group nil lists and no description",
			ns:   []netbird.NameserverGroup{{ID: "n1", Name: "bare"}},
			want: `{"nameserver_groups":[{"id":"n1","name":"bare","enabled":false,"primary":false,"domains":[],` +
				`"search_domains_enabled":false,"groups":[],"nameservers":[]}],"zones":[],"disabled_management_groups":[]}`,
		},
		{
			name: "zone all fields",
			zones: []netbird.DNSZone{{
				ID: "z1", Name: "internal", Domain: "corp.example", Enabled: true, EnableSearchDomain: true,
				DistributionGroups: []string{"g1"},
				Records: []netbird.DNSRecord{
					{ID: "r1", Name: "a.corp.example", Type: "A", Content: "10.0.0.1", TTL: 300},
					{ID: "r2", Name: "b.corp.example", Type: "CNAME", Content: "a.corp.example", TTL: 60},
				},
			}},
			groups: groups,
			want: `{"nameserver_groups":[],"zones":[{"id":"z1","name":"internal","domain":"corp.example","enabled":true,` +
				`"enable_search_domain":true,"distribution_groups":[{"id":"g1","name":"devs"}],"records":[` +
				`{"id":"r1","name":"a.corp.example","type":"A","content":"10.0.0.1","ttl":300},` +
				`{"id":"r2","name":"b.corp.example","type":"CNAME","content":"a.corp.example","ttl":60}]}],"disabled_management_groups":[]}`,
		},
		{
			name:  "zone nil lists",
			zones: []netbird.DNSZone{{ID: "z1", Name: "bare", Domain: "x.example"}},
			want: `{"nameserver_groups":[],"zones":[{"id":"z1","name":"bare","domain":"x.example","enabled":false,` +
				`"enable_search_domain":false,"distribution_groups":[],"records":[]}],"disabled_management_groups":[]}`,
		},
		{
			name:     "disabled management groups keep order",
			settings: netbird.DNSSettings{DisabledManagementGroups: []string{"g2", "g1"}},
			groups:   groups,
			want:     `{"nameserver_groups":[],"zones":[],"disabled_management_groups":[{"id":"g2","name":"ops"},{"id":"g1","name":"devs"}]}`,
		},
		{
			name:     "unknown group ids have no name",
			ns:       []netbird.NameserverGroup{{ID: "n1", Name: "x", Groups: []string{"gone", "g1"}}},
			zones:    []netbird.DNSZone{{ID: "z1", DistributionGroups: []string{"gone"}}},
			settings: netbird.DNSSettings{DisabledManagementGroups: []string{"gone"}},
			groups:   groups,
			want: `{"nameserver_groups":[{"id":"n1","name":"x","enabled":false,"primary":false,"domains":[],"search_domains_enabled":false,` +
				`"groups":[{"id":"gone"},{"id":"g1","name":"devs"}],"nameservers":[]}],` +
				`"zones":[{"id":"z1","name":"","domain":"","enabled":false,"enable_search_domain":false,"distribution_groups":[{"id":"gone"}],"records":[]}],` +
				`"disabled_management_groups":[{"id":"gone"}]}`,
		},
		{
			name:  "order kept",
			ns:    []netbird.NameserverGroup{{ID: "n2", Name: "b"}, {ID: "n1", Name: "a"}},
			zones: []netbird.DNSZone{{ID: "z2"}, {ID: "z1"}},
			want: `{"nameserver_groups":[` +
				`{"id":"n2","name":"b","enabled":false,"primary":false,"domains":[],"search_domains_enabled":false,"groups":[],"nameservers":[]},` +
				`{"id":"n1","name":"a","enabled":false,"primary":false,"domains":[],"search_domains_enabled":false,"groups":[],"nameservers":[]}],` +
				`"zones":[` +
				`{"id":"z2","name":"","domain":"","enabled":false,"enable_search_domain":false,"distribution_groups":[],"records":[]},` +
				`{"id":"z1","name":"","domain":"","enabled":false,"enable_search_domain":false,"distribution_groups":[],"records":[]}],` +
				`"disabled_management_groups":[]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := marshalDNSOutput(t, tt.ns, tt.settings, tt.zones, tt.groups)
			if got != tt.want {
				t.Errorf("output =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestListDNSTool(t *testing.T) {
	t.Parallel()
	dns := &fakeDNSAPI{
		nsGroups: []netbird.NameserverGroup{{
			ID: "n1", Name: "cf", Enabled: true, Groups: []string{"g1"},
			Nameservers: []netbird.Nameserver{{IP: "1.1.1.1", NSType: "udp", Port: 53}},
		}},
		settings: netbird.DNSSettings{DisabledManagementGroups: []string{"g2"}},
		zones: []netbird.DNSZone{{
			ID: "z1", Name: "internal", Domain: "corp.example", DistributionGroups: []string{"g1"},
			Records: []netbird.DNSRecord{{ID: "r1", Name: "a.corp.example", Type: "A", Content: "10.0.0.1", TTL: 300}},
		}},
	}
	grp := &fakeGroupAPI{groups: []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "ops"}}}
	cs, logs := connectFake(t, fakeClient{dnsAPI: dns, groupAPI: grp})
	res, text := callTool(t, cs, "list_dns", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	const want = `{"nameserver_groups":[{"id":"n1","name":"cf","enabled":true,"primary":false,"domains":[],"search_domains_enabled":false,` +
		`"groups":[{"id":"g1","name":"devs"}],"nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53}]}],` +
		`"zones":[{"id":"z1","name":"internal","domain":"corp.example","enabled":false,"enable_search_domain":false,` +
		`"distribution_groups":[{"id":"g1","name":"devs"}],"records":[{"id":"r1","name":"a.corp.example","type":"A","content":"10.0.0.1","ttl":300}]}],` +
		`"disabled_management_groups":[{"id":"g2","name":"ops"}]}`
	var gotV, wantV any
	if err := json.Unmarshal(raw, &gotV); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gotV, wantV) {
		t.Errorf("structured =\n%s\nwant\n%s", raw, want)
	}
	if dns.calls.Load() != 3 || grp.calls != 1 {
		t.Errorf("calls = %d/%d, want 3/1", dns.calls.Load(), grp.calls)
	}
	if !strings.Contains(logs.String(), "tool=list_dns") || !strings.Contains(logs.String(), "is_error=false") {
		t.Errorf("logs missing tool fields: %s", logs)
	}
	if strings.Contains(text, sentinelToken) || strings.Contains(string(raw), sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
		t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
	}
}

func TestListDNSToolEmpty(t *testing.T) {
	t.Parallel()
	cs, _ := connectFake(t, fakeClient{dnsAPI: &fakeDNSAPI{}, groupAPI: &fakeGroupAPI{}})
	res, text := callTool(t, cs, "list_dns", nil)
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	sc, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content = %#v", res.StructuredContent)
	}
	for _, key := range []string{"nameserver_groups", "zones", "disabled_management_groups"} {
		list, ok := sc[key].([]any)
		if !ok || len(list) != 0 {
			t.Errorf("%s = %#v, want empty non-nil", key, sc[key])
		}
	}
}

func TestListDNSToolDegrades(t *testing.T) {
	t.Parallel()
	nsGroups := []netbird.NameserverGroup{{
		ID: "n1", Name: "cf", Enabled: true, Groups: []string{"g1"},
		Nameservers: []netbird.Nameserver{{IP: "1.1.1.1", NSType: "udp", Port: 53}},
	}}
	zones := []netbird.DNSZone{{ID: "z1", Name: "internal", Domain: "corp.example", DistributionGroups: []string{"g1"}}}
	settings := netbird.DNSSettings{DisabledManagementGroups: []string{"g2"}}
	groups := []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "ops"}}
	tests := []struct {
		name string
		dns  *fakeDNSAPI
		grp  *fakeGroupAPI
		want string
	}{
		{
			name: "groups failure keeps ids without names",
			dns:  &fakeDNSAPI{nsGroups: nsGroups, settings: settings, zones: zones},
			grp:  &fakeGroupAPI{err: &netbird.APIError{Status: 500, Message: "boom"}},
			want: `{"nameserver_groups":[{"id":"n1","name":"cf","enabled":true,"primary":false,"domains":[],"search_domains_enabled":false,` +
				`"groups":[{"id":"g1"}],"nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53}]}],` +
				`"zones":[{"id":"z1","name":"internal","domain":"corp.example","enabled":false,"enable_search_domain":false,` +
				`"distribution_groups":[{"id":"g1"}],"records":[]}],` +
				`"disabled_management_groups":[{"id":"g2"}],` +
				`"group_names_error":"netbird api: 500: boom (hint: NetBird server error, retry later)"}`,
		},
		{
			name: "zones failure gives empty zones",
			dns:  &fakeDNSAPI{nsGroups: nsGroups, settings: settings, zonesErr: &netbird.APIError{Status: 404, Message: "no zones"}},
			grp:  &fakeGroupAPI{groups: groups},
			want: `{"nameserver_groups":[{"id":"n1","name":"cf","enabled":true,"primary":false,"domains":[],"search_domains_enabled":false,` +
				`"groups":[{"id":"g1","name":"devs"}],"nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53}]}],` +
				`"zones":[],"disabled_management_groups":[{"id":"g2","name":"ops"}],` +
				`"zones_error":"ZONES_ERR"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{dnsAPI: tt.dns, groupAPI: tt.grp})
			res, text := callTool(t, cs, "list_dns", nil)
			if res.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			raw, err := json.Marshal(res.StructuredContent)
			if err != nil {
				t.Fatalf("marshal result: %v", err)
			}
			want := tt.want
			if tt.dns.zonesErr != nil {
				want = strings.Replace(want, "ZONES_ERR", toolMessage(tt.dns.zonesErr), 1)
			}
			var gotV, wantV any
			if err := json.Unmarshal(raw, &gotV); err != nil {
				t.Fatalf("unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(want), &wantV); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}
			if !reflect.DeepEqual(gotV, wantV) {
				t.Errorf("structured =\n%s\nwant\n%s", raw, want)
			}
			if !strings.Contains(logs.String(), "is_error=false") {
				t.Errorf("logs missing is_error=false: %s", logs)
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(string(raw), sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q result=%s logs=%s", text, raw, logs)
			}
		})
	}
}

func TestListDNSToolErrors(t *testing.T) {
	t.Parallel()
	unauthorized := &netbird.APIError{Status: 401, Message: "bad token"}
	const want401 = "netbird api: 401: bad token (hint: token invalid or expired)"
	tests := []struct {
		name    string
		dns     *fakeDNSAPI
		grp     *fakeGroupAPI
		wantErr string
	}{
		{name: "nameserver groups fail with 401", dns: &fakeDNSAPI{nsErr: unauthorized}, grp: &fakeGroupAPI{}, wantErr: want401},
		{name: "settings fail with 401", dns: &fakeDNSAPI{settingsErr: unauthorized}, grp: &fakeGroupAPI{}, wantErr: want401},
		{
			name:    "settings fail with 503",
			dns:     &fakeDNSAPI{settingsErr: &netbird.APIError{Status: 503, Message: "down"}},
			grp:     &fakeGroupAPI{},
			wantErr: "netbird api: 503: down (hint: NetBird server error, retry later)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{dnsAPI: tt.dns, groupAPI: tt.grp})
			res, text := callTool(t, cs, "list_dns", nil)
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

func TestListDNSToolDefinition(t *testing.T) {
	t.Parallel()
	tool, _ := listDNS(struct {
		dnsAPI
		groupAPI
	}{&fakeDNSAPI{}, &fakeGroupAPI{}})
	if tool.Name != "list_dns" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Lists the NetBird DNS setup: nameserver groups (nameservers, match domains or primary, the peer groups using them), custom DNS zones with their A, AAAA and CNAME records and distribution groups, and the groups whose DNS management is disabled. If zones or group names cannot be fetched, zones is empty or names are missing, and zones_error or group_names_error explains why."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
	cs, _ := connectFake(t, fakeClient{dnsAPI: &fakeDNSAPI{}, groupAPI: &fakeGroupAPI{}})
	list, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	for _, lt := range list.Tools {
		if lt.Name != "list_dns" {
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
	t.Error("list_dns not registered")
}
