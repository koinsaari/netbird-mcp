package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

const networkFanOut = 4

type networkAPI interface {
	ListNetworks(ctx context.Context) ([]netbird.Network, error)
	ListNetworkRouters(ctx context.Context, networkID string) ([]netbird.NetworkRouter, error)
	ListNetworkResources(ctx context.Context, networkID string) ([]netbird.NetworkResource, error)
}

type listNetworksAPI interface {
	networkAPI
	groupAPI
	peerAPI
}

type listNetworksInput struct{}

type listNetworksOutput struct {
	Networks   []networkOutput `json:"networks"`
	NamesError string          `json:"names_error,omitempty"`
}

type networkOutput struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description,omitempty"`
	RoutingPeersCount int                     `json:"routing_peers_count"`
	Policies          []string                `json:"policies"`
	Routers           []networkRouterOutput   `json:"routers"`
	Resources         []networkResourceOutput `json:"resources"`
}

type networkRouterOutput struct {
	ID         string         `json:"id"`
	Peer       *peerRefOutput `json:"peer,omitempty"`
	PeerGroups []groupOutput  `json:"peer_groups"`
	Metric     int            `json:"metric"`
	Masquerade bool           `json:"masquerade"`
	Enabled    bool           `json:"enabled"`
}

type networkResourceOutput struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Address     string        `json:"address"`
	Type        string        `json:"type"`
	Enabled     bool          `json:"enabled"`
	Groups      []groupOutput `json:"groups"`
}

func listNetworks(c listNetworksAPI) (*mcp.Tool, mcp.ToolHandlerFor[listNetworksInput, listNetworksOutput]) {
	tool := &mcp.Tool{
		Name:        "list_networks",
		Description: "Lists all NetBird networks with their routers (routing peer or peer groups, metric, masquerade) and resources (address, type, groups). Disabled routers and resources are included with their enabled flag. Group and peer references include names; names_error appears when names cannot be fetched.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listNetworksInput) (*mcp.CallToolResult, listNetworksOutput, error) {
		var (
			networks  []netbird.Network
			routers   [][]netbird.NetworkRouter
			resources [][]netbird.NetworkResource
		)
		names, err := resolveNames(ctx, c, c, func(ctx context.Context) (err error) {
			networks, routers, resources, err = fetchNetworks(ctx, c)
			return err
		})
		if err != nil {
			return nil, listNetworksOutput{}, err
		}
		out := toListNetworksOutput(networks, routers, resources, names.groups, names.peers)
		out.NamesError = names.err
		return nil, out, nil
	}
	return tool, handler
}

func fetchNetworks(ctx context.Context, c networkAPI) ([]netbird.Network, [][]netbird.NetworkRouter, [][]netbird.NetworkResource, error) {
	networks, err := c.ListNetworks(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	routers := make([][]netbird.NetworkRouter, len(networks))
	resources := make([][]netbird.NetworkResource, len(networks))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(networkFanOut)
	for i, n := range networks {
		if len(n.Routers) > 0 {
			g.Go(func() (err error) {
				if routers[i], err = c.ListNetworkRouters(gctx, n.ID); err != nil {
					return fmt.Errorf("network %s: %w", n.ID, err)
				}
				return nil
			})
		}
		if len(n.Resources) > 0 {
			g.Go(func() (err error) {
				if resources[i], err = c.ListNetworkResources(gctx, n.ID); err != nil {
					return fmt.Errorf("network %s: %w", n.ID, err)
				}
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return nil, nil, nil, err
	}
	return networks, routers, resources, nil
}

func toListNetworksOutput(networks []netbird.Network, routers [][]netbird.NetworkRouter, resources [][]netbird.NetworkResource, groupNames, peerNames map[string]string) listNetworksOutput {
	out := listNetworksOutput{Networks: make([]networkOutput, 0, len(networks))}
	for i, n := range networks {
		no := networkOutput{
			ID: n.ID, Name: n.Name, Description: n.Description, RoutingPeersCount: n.RoutingPeersCount,
			Policies:  nonNil(n.Policies),
			Routers:   make([]networkRouterOutput, 0, len(routers[i])),
			Resources: make([]networkResourceOutput, 0, len(resources[i])),
		}
		for _, r := range routers[i] {
			no.Routers = append(no.Routers, networkRouterOutput{
				ID: r.ID, Peer: toPeerRef(r.Peer, peerNames), PeerGroups: toGroupRefs(r.PeerGroups, groupNames),
				Metric: r.Metric, Masquerade: r.Masquerade, Enabled: r.Enabled,
			})
		}
		for _, r := range resources[i] {
			no.Resources = append(no.Resources, networkResourceOutput{
				ID: r.ID, Name: r.Name, Description: r.Description, Address: r.Address,
				Type: r.Type, Enabled: r.Enabled, Groups: toGroupOutputs(r.Groups),
			})
		}
		out.Networks = append(out.Networks, no)
	}
	return out
}
