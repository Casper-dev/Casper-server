package mdutils

import (
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	dssync "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore/sync"

	"github.com/Casper-dev/Casper-server/blocks/blockstore"
	bsrv "github.com/Casper-dev/Casper-server/blockservice"
	"github.com/Casper-dev/Casper-server/exchange/offline"
	dag "github.com/Casper-dev/Casper-server/merkledag"
)

func Mock() dag.DAGService {
	return dag.NewDAGService(Bserv())
}

func Bserv() bsrv.BlockService {
	bstore := blockstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
	return bsrv.New(bstore, offline.Exchange(bstore))
}
