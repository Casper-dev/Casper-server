package config

import scin "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"

// TODO remove after alpha
const DefaultETHPrivateKey = "674393e0fb1cba8a71be3f1261e7171effb998bc5047ae0eee8b0e49e556e293"
const DefaultETHGateway = "http://94.130.182.144:8775"

var DefaultETHOpts = scin.InitOpts{
	"PrivateKey": DefaultETHPrivateKey,
	"Gateway":    DefaultETHGateway,
}

const DefaultNEOWif = "KxDgvEKzgSBPPfuVfw67oPQBSjidEiqTHURKSDL1R7yGaGYAeYnr"
const DefaultNEOGateway = "http://127.0.0.1:10332"
const DefaultNeonAPI = "127.0.0.1:5000"

var DefaultNEOOpts = scin.InitOpts{
	"WIF":     DefaultNEOWif,
	"Gateway": DefaultNEOGateway,
	"NeonAPI": DefaultNeonAPI,
}

var DefaultMULTIOpts = scin.InitOpts{
	"NEO": DefaultNEOOpts,
	"ETH": DefaultETHOpts,
}

type Casper struct {
	DiskSizeBytes   int64
	IPAddress       string
	TelegramAddress string
	ConnectionPort  string
	Blockchain      map[string]scin.InitOpts
	UsedChain       string
}
