package tools

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type refNames struct {
	groups map[string]string
	peers  map[string]string
	err    string
}

func resolveNames(ctx context.Context, g groupAPI, p peerAPI, core func(context.Context) error) (refNames, error) {
	var (
		groups              []netbird.Group
		peers               []netbird.Peer
		groupsErr, peersErr error
	)
	eg, gctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return core(gctx)
	})
	eg.Go(func() error {
		groups, groupsErr = g.ListGroups(gctx, netbird.ListGroupsParams{})
		return nil
	})
	if p != nil {
		eg.Go(func() error {
			peers, peersErr = p.ListPeers(gctx, netbird.ListPeersParams{})
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return refNames{}, err
	}
	var names refNames
	switch {
	case groupsErr != nil:
		names.err = toolMessage(groupsErr)
	case peersErr != nil:
		names.err = toolMessage(peersErr)
	}
	if groupsErr != nil {
		groups = nil
	}
	if peersErr != nil {
		peers = nil
	}
	names.groups = groupNamesByID(groups)
	names.peers = peerNamesByID(peers)
	return names, nil
}
