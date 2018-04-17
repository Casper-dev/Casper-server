package mdutils

import (
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	dssync "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore/sync"

	"gitlab.com/casperDev/Casper-server/blocks/blockstore"
	bsrv "gitlab.com/casperDev/Casper-server/blockservice"
	"gitlab.com/casperDev/Casper-server/exchange/offline"
	dag "gitlab.com/casperDev/Casper-server/merkledag"
)

func Mock() dag.DAGService {
	return dag.NewDAGService(Bserv())
}

func Bserv() bsrv.BlockService {
	bstore := blockstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
	return bsrv.New(bstore, offline.Exchange(bstore))
}
