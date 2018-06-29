package sc

import (
	"context"

	multi "github.com/Casper-dev/Casper-server/casper/sc/multichain"
	neo "github.com/Casper-dev/Casper-server/casper/sc/neo"
	scin "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"
	sol "github.com/Casper-dev/Casper-server/casper/sc/solidity"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

var log = logging.Logger("sc")
var contracts = map[string]scin.CasperSC{
	Ethereum: &sol.Contract{},
	NEO:      &neo.Contract{},
	Multi:    &multi.Contract{},
}

var argsCache = make(map[string]scin.InitOpts, 2)

const (
	Ethereum     = sol.ChainName
	NEO          = neo.ChainName
	Multi        = multi.ChainName
	DefaultChain = Ethereum
)

func GetContract(args ...interface{}) (scin.CasperSC, error) {
	return GetContractContext(context.Background(), args...)
}

func GetContractContext(ctx context.Context, args ...interface{}) (scin.CasperSC, error) {
	return GetContractByName(ctx, DefaultChain, args...)
}

func GetContractByName(ctx context.Context, name string, args ...interface{}) (c scin.CasperSC, err error) {
	log.Debugf("name=%s, args=%+v", name, args)
	c = contracts[name]
	if len(args) > 0 {
		argsCache[name] = args[0].(scin.InitOpts)
	}
	log.Debugf("sc already sinitialized: %t", c.Initialized())
	if !c.Initialized() {
		err = c.Init(ctx, argsCache[name])
		if err != nil {
			log.Errorf("error while initializing SC: %v", err)
		}
	}

	return c, err
}
