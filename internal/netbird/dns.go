package netbird

import (
	"context"
	"fmt"
)

type Nameserver struct {
	IP     string `json:"ip"`
	NSType string `json:"ns_type"`
	Port   int    `json:"port"`
}

type NameserverGroup struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	Nameservers          []Nameserver `json:"nameservers"`
	Enabled              bool         `json:"enabled"`
	Groups               []string     `json:"groups"`
	Primary              bool         `json:"primary"`
	Domains              []string     `json:"domains"`
	SearchDomainsEnabled bool         `json:"search_domains_enabled"`
}

type DNSSettings struct {
	DisabledManagementGroups []string `json:"disabled_management_groups"`
}

type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type DNSZone struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Domain             string      `json:"domain"`
	Enabled            bool        `json:"enabled"`
	EnableSearchDomain bool        `json:"enable_search_domain"`
	DistributionGroups []string    `json:"distribution_groups"`
	Records            []DNSRecord `json:"records"`
}

func (c *Client) ListNameserverGroups(ctx context.Context) ([]NameserverGroup, error) {
	var groups []NameserverGroup
	if err := c.get(ctx, &groups, nil, "dns", "nameservers"); err != nil {
		return nil, fmt.Errorf("list nameserver groups: %w", err)
	}
	return groups, nil
}

func (c *Client) GetDNSSettings(ctx context.Context) (DNSSettings, error) {
	var settings DNSSettings
	if err := c.get(ctx, &settings, nil, "dns", "settings"); err != nil {
		return DNSSettings{}, fmt.Errorf("get dns settings: %w", err)
	}
	return settings, nil
}

func (c *Client) ListDNSZones(ctx context.Context) ([]DNSZone, error) {
	var zones []DNSZone
	if err := c.get(ctx, &zones, nil, "dns", "zones"); err != nil {
		return nil, fmt.Errorf("list dns zones: %w", err)
	}
	return zones, nil
}
