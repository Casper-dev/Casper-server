package bitswap

import (
	"context"
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	testutil "gx/ipfs/QmWRCn8vruNAzHx8i6SAXinuheRitKEGu8c7m26stKvsYx/go-testutil"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	mockpeernet "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/net/mock"

	bsnet "github.com/Casper-dev/Casper-server/exchange/bitswap/network"
	mockrouting "github.com/Casper-dev/Casper-server/routing/mock"
)

type peernet struct {
	mockpeernet.Mocknet
	routingserver mockrouting.Server
}

func StreamNet(ctx context.Context, net mockpeernet.Mocknet, rs mockrouting.Server) (Network, error) {
	return &peernet{net, rs}, nil
}

func (pn *peernet) Adapter(p testutil.Identity) bsnet.BitSwapNetwork {
	client, err := pn.Mocknet.AddPeer(p.PrivateKey(), p.Address())
	if err != nil {
		panic(err.Error())
	}
	routing := pn.routingserver.ClientWithDatastore(context.TODO(), p, ds.NewMapDatastore())
	return bsnet.NewFromIpfsHost(client, routing)
}

func (pn *peernet) HasPeer(p peer.ID) bool {
	for _, member := range pn.Mocknet.Peers() {
		if p == member {
			return true
		}
	}
	return false
}

var _ Network = (*peernet)(nil)
