package bstest

import (
	. "gitlab.com/casperDev/Casper-server/blockservice"
	bitswap "gitlab.com/casperDev/Casper-server/exchange/bitswap"
	tn "gitlab.com/casperDev/Casper-server/exchange/bitswap/testnet"
	mockrouting "gitlab.com/casperDev/Casper-server/routing/mock"
	delay "gitlab.com/casperDev/Casper-server/thirdparty/delay"
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
