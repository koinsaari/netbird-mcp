package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type policyAPI interface {
	ListPolicies(ctx context.Context) ([]netbird.Policy, error)
}

type listPoliciesInput struct{}

type listPoliciesOutput struct {
	Policies []policyOutput `json:"policies"`
}

type policyOutput struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Description         string             `json:"description,omitempty"`
	Enabled             bool               `json:"enabled"`
	SourcePostureChecks []string           `json:"source_posture_checks"`
	Rules               []policyRuleOutput `json:"rules"`
}

type policyRuleOutput struct {
	ID                  string              `json:"id,omitempty"`
	Name                string              `json:"name"`
	Description         string              `json:"description,omitempty"`
	Enabled             bool                `json:"enabled"`
	Action              string              `json:"action"`
	Bidirectional       bool                `json:"bidirectional"`
	Protocol            string              `json:"protocol"`
	Ports               []string            `json:"ports,omitempty"`
	PortRanges          []portRangeOutput   `json:"port_ranges,omitempty"`
	Sources             []ruleGroupOutput   `json:"sources"`
	SourceResource      *ruleResourceOutput `json:"source_resource,omitempty"`
	Destinations        []ruleGroupOutput   `json:"destinations"`
	DestinationResource *ruleResourceOutput `json:"destination_resource,omitempty"`
	AuthorizedGroups    map[string][]string `json:"authorized_groups,omitempty"`
}

type portRangeOutput struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ruleGroupOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ruleResourceOutput struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// listPolicies builds the list_policies tool definition and its handler.
func listPolicies(c policyAPI) (*mcp.Tool, mcp.ToolHandlerFor[listPoliciesInput, listPoliciesOutput]) {
	tool := &mcp.Tool{
		Name:        "list_policies",
		Description: "Lists all access control policies in the NetBird account with their rules: sources, destinations, protocol, ports and action.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listPoliciesInput) (*mcp.CallToolResult, listPoliciesOutput, error) {
		policies, err := c.ListPolicies(ctx)
		if err != nil {
			return nil, listPoliciesOutput{}, err
		}
		return nil, toListPoliciesOutput(policies), nil
	}
	return tool, handler
}

func toListPoliciesOutput(policies []netbird.Policy) listPoliciesOutput {
	out := listPoliciesOutput{Policies: make([]policyOutput, 0, len(policies))}
	for _, p := range policies {
		checks := make([]string, 0, len(p.SourcePostureChecks))
		checks = append(checks, p.SourcePostureChecks...)
		rules := make([]policyRuleOutput, 0, len(p.Rules))
		for _, r := range p.Rules {
			rules = append(rules, toPolicyRuleOutput(r))
		}
		out.Policies = append(out.Policies, policyOutput{
			ID: p.ID, Name: p.Name, Description: p.Description, Enabled: p.Enabled,
			SourcePostureChecks: checks, Rules: rules,
		})
	}
	return out
}

func toPolicyRuleOutput(r netbird.PolicyRule) policyRuleOutput {
	var ranges []portRangeOutput
	for _, pr := range r.PortRanges {
		ranges = append(ranges, portRangeOutput{Start: pr.Start, End: pr.End})
	}
	return policyRuleOutput{
		ID: r.ID, Name: r.Name, Description: r.Description, Enabled: r.Enabled, Action: r.Action,
		Bidirectional: r.Bidirectional, Protocol: r.Protocol, Ports: r.Ports, PortRanges: ranges,
		Sources: toRuleGroups(r.Sources), SourceResource: toRuleResource(r.SourceResource),
		Destinations: toRuleGroups(r.Destinations), DestinationResource: toRuleResource(r.DestinationResource),
		AuthorizedGroups: r.AuthorizedGroups,
	}
}

func toRuleGroups(groups []netbird.Group) []ruleGroupOutput {
	out := make([]ruleGroupOutput, 0, len(groups))
	for _, g := range groups {
		out = append(out, ruleGroupOutput{ID: g.ID, Name: g.Name})
	}
	return out
}

func toRuleResource(r *netbird.Resource) *ruleResourceOutput {
	if r == nil {
		return nil
	}
	return &ruleResourceOutput{ID: r.ID, Type: r.Type}
}
