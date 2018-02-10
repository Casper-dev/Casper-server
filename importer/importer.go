// package importer implements utilities used to create IPFS DAGs from files
// and readers
package importer

import (
	"fmt"
	"os"

	"github.com/Casper-dev/Casper-server/commands/files"
	bal "github.com/Casper-dev/Casper-server/importer/balanced"
	"github.com/Casper-dev/Casper-server/importer/chunk"
	h "github.com/Casper-dev/Casper-server/importer/helpers"
	trickle "github.com/Casper-dev/Casper-server/importer/trickle"
	dag "github.com/Casper-dev/Casper-server/merkledag"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
)

// Builds a DAG from the given file, writing created blocks to disk as they are
// created
func BuildDagFromFile(fpath string, ds dag.DAGService) (node.Node, error) {
	stat, err := os.Lstat(fpath)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("`%s` is a directory", fpath)
	}

	f, err := files.NewSerialFile(fpath, fpath, false, stat)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return BuildDagFromReader(ds, chunk.NewSizeSplitter(f, chunk.DefaultBlockSize))
}

func BuildDagFromReader(ds dag.DAGService, spl chunk.Splitter) (node.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return bal.BalancedLayout(dbp.New(spl))
}

func BuildTrickleDagFromReader(ds dag.DAGService, spl chunk.Splitter) (node.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return trickle.TrickleLayout(dbp.New(spl))
}
