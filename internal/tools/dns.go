package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type dnsAPI interface {
	ListNameserverGroups(ctx context.Context) ([]netbird.NameserverGroup, error)
	GetDNSSettings(ctx context.Context) (netbird.DNSSettings, error)
	ListDNSZones(ctx context.Context) ([]netbird.DNSZone, error)
}

type listDNSAPI interface {
	dnsAPI
	groupAPI
}

type listDNSInput struct{}

type listDNSOutput struct {
	NameserverGroups         []nameserverGroupOutput `json:"nameserver_groups"`
	Zones                    []dnsZoneOutput         `json:"zones"`
	DisabledManagementGroups []groupOutput           `json:"disabled_management_groups"`
	ZonesError               string                  `json:"zones_error,omitempty"`
	GroupNamesError          string                  `json:"group_names_error,omitempty"`
}

type nameserverGroupOutput struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description,omitempty"`
	Enabled              bool               `json:"enabled"`
	Primary              bool               `json:"primary"`
	Domains              []string           `json:"domains"`
	SearchDomainsEnabled bool               `json:"search_domains_enabled"`
	Groups               []groupOutput      `json:"groups"`
	Nameservers          []nameserverOutput `json:"nameservers"`
}

type nameserverOutput struct {
	IP     string `json:"ip"`
	NSType string `json:"ns_type"`
	Port   int    `json:"port"`
}

type dnsZoneOutput struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Domain             string            `json:"domain"`
	Enabled            bool              `json:"enabled"`
	EnableSearchDomain bool              `json:"enable_search_domain"`
	DistributionGroups []groupOutput     `json:"distribution_groups"`
	Records            []dnsRecordOutput `json:"records"`
}

type dnsRecordOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

func listDNS(c listDNSAPI) (*mcp.Tool, mcp.ToolHandlerFor[listDNSInput, listDNSOutput]) {
	tool := &mcp.Tool{
		Name:        "list_dns",
		Description: "Lists the NetBird DNS setup: nameserver groups (nameservers, match domains or primary, the peer groups using them), custom DNS zones with their A, AAAA and CNAME records and distribution groups, and the groups whose DNS management is disabled. If zones or group names cannot be fetched, zones is empty or names are missing, and zones_error or group_names_error explains why.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listDNSInput) (*mcp.CallToolResult, listDNSOutput, error) {
		var (
			nsGroups []netbird.NameserverGroup
			settings netbird.DNSSettings
			zones    []netbird.DNSZone
			groups   []netbird.Group

			zonesErr, groupNamesErr string
		)
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() (err error) {
			nsGroups, err = c.ListNameserverGroups(gctx)
			return err
		})
		g.Go(func() (err error) {
			settings, err = c.GetDNSSettings(gctx)
			return err
		})
		g.Go(func() error {
			var zerr error
			zones, zerr = c.ListDNSZones(gctx)
			if zerr != nil {
				zones = nil
				zonesErr = toolMessage(zerr)
			}
			return nil
		})
		g.Go(func() error {
			var gerr error
			groups, gerr = c.ListGroups(gctx, netbird.ListGroupsParams{})
			if gerr != nil {
				groups = nil
				groupNamesErr = toolMessage(gerr)
			}
			return nil
		})
		if err := g.Wait(); err != nil {
			return nil, listDNSOutput{}, err
		}
		out := toListDNSOutput(nsGroups, settings, zones, groups)
		out.ZonesError = zonesErr
		out.GroupNamesError = groupNamesErr
		return nil, out, nil
	}
	return tool, handler
}

func toListDNSOutput(nsGroups []netbird.NameserverGroup, settings netbird.DNSSettings, zones []netbird.DNSZone, groups []netbird.Group) listDNSOutput {
	names := groupNamesByID(groups)
	out := listDNSOutput{
		NameserverGroups:         make([]nameserverGroupOutput, 0, len(nsGroups)),
		Zones:                    make([]dnsZoneOutput, 0, len(zones)),
		DisabledManagementGroups: toGroupRefs(settings.DisabledManagementGroups, names),
	}
	for _, ns := range nsGroups {
		servers := make([]nameserverOutput, 0, len(ns.Nameservers))
		for _, s := range ns.Nameservers {
			servers = append(servers, nameserverOutput{IP: s.IP, NSType: s.NSType, Port: s.Port})
		}
		out.NameserverGroups = append(out.NameserverGroups, nameserverGroupOutput{
			ID: ns.ID, Name: ns.Name, Description: ns.Description, Enabled: ns.Enabled, Primary: ns.Primary,
			Domains: nonNil(ns.Domains), SearchDomainsEnabled: ns.SearchDomainsEnabled,
			Groups: toGroupRefs(ns.Groups, names), Nameservers: servers,
		})
	}
	for _, z := range zones {
		records := make([]dnsRecordOutput, 0, len(z.Records))
		for _, r := range z.Records {
			records = append(records, dnsRecordOutput{ID: r.ID, Name: r.Name, Type: r.Type, Content: r.Content, TTL: r.TTL})
		}
		out.Zones = append(out.Zones, dnsZoneOutput{
			ID: z.ID, Name: z.Name, Domain: z.Domain, Enabled: z.Enabled, EnableSearchDomain: z.EnableSearchDomain,
			DistributionGroups: toGroupRefs(z.DistributionGroups, names), Records: records,
		})
	}
	return out
}
