package bstest

import (
	. "github.com/Casper-dev/Casper-server/blockservice"
	bitswap "github.com/Casper-dev/Casper-server/exchange/bitswap"
	tn "github.com/Casper-dev/Casper-server/exchange/bitswap/testnet"
	mockrouting "github.com/Casper-dev/Casper-server/routing/mock"
	delay "github.com/Casper-dev/Casper-server/thirdparty/delay"
)

// Mocks returns |n| connected mock Blockservices
func Mocks(n int) []BlockService {
	net := tn.VirtualNetwork(mockrouting.NewServer(), delay.Fixed(0))
	sg := bitswap.NewTestSessionGenerator(net)

	instances := sg.Instances(n)

	var servs []BlockService
	for _, i := range instances {
		servs = append(servs, New(i.Blockstore(), i.Exchange))
	}
	return servs
}
