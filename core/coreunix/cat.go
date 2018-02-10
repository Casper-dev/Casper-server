package coreunix

import (
	"context"

	core "github.com/Casper-dev/Casper-server/core"
	path "github.com/Casper-dev/Casper-server/path"
	uio "github.com/Casper-dev/Casper-server/unixfs/io"
)

func Cat(ctx context.Context, n *core.IpfsNode, pstr string) (uio.DagReader, error) {
	r := &path.Resolver{
		DAG:         n.DAG,
		ResolveOnce: uio.ResolveUnixfsOnce,
	}

	dagNode, err := core.Resolve(ctx, n.Namesys, r, path.Path(pstr))
	if err != nil {
		return nil, err
	}

	return uio.NewDagReader(ctx, dagNode, n.DAG)
}
