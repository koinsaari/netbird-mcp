package tools

import "github.com/koinsaari/netbird-mcp/internal/netbird"

type groupOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toGroupOutputs(groups []netbird.Group) []groupOutput {
	out := make([]groupOutput, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupOutput{ID: g.ID, Name: g.Name})
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
