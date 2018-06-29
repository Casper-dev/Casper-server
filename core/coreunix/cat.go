package coreunix

import (
	"context"

	"fmt"

	core "github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/merkledag"
	path "github.com/Casper-dev/Casper-server/path"
	ft "github.com/Casper-dev/Casper-server/unixfs"
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
	fmt.Println(dagNode)
	fmt.Println(err)

	if v, ok := dagNode.(*merkledag.ProtoNode); ok {
		if fsn, err := ft.FSNodeFromBytes(v.Data()); err == nil {
			if fsn.Type == ft.TDirectory && len(fsn.Data) == 0 && len(v.Links()) == 1 {
				dagNode, _ = v.Links()[0].GetNode(ctx, n.DAG)
			}
		}
	}

	return uio.NewDagReader(ctx, dagNode, n.DAG)
}
