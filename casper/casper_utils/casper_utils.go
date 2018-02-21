package casper_utils

import (
	"fmt"

	"math/big"
	"git.apache.org/thrift.git/lib/go/thrift"
	"crypto/tls"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-SC/casper_sc"


	"github.com/Casper-dev/Casper-server/repo/config"
	"regexp"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/Casper-dev/Casper-SC/casperproto"
)

var fullNodeID string

func RegisterSC(node *core.IpfsNode, cfg *config.Config) {
	fmt.Println(node.Identity.Pretty())
	casper, client, auth := Casper_SC.GetSC()
	ip := Casper_SC.GetIPv4()
	re := regexp.MustCompile("/tcp/.+")
	port := re.FindString(cfg.Addresses.Swarm[0])
	fullNodeID = "/ip4/" + ip + port + "/ipfs/" + node.Identity.Pretty()
	fmt.Println(fullNodeID)

	///TODO: debug only: remove as soon as we deploy
	addTokenClosure := func() (*types.Transaction, error) {
		return casper.AddToken(auth, big.NewInt(int64(13370000000)))
	}
	Casper_SC.ValidateMineTX(addTokenClosure, client)
	fmt.Println("registering")

	registerProviderClosure := func() (*types.Transaction, error) {
		return casper.RegisterProvider(auth, fullNodeID, big.NewInt(int64(13370000000)))
	}

	Casper_SC.ValidateMineTX(registerProviderClosure, client)

	return
}


func GetCasperNodeID() string {
	return fullNodeID
}

var protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()

var transportFactory = thrift.NewTBufferedTransportFactory(8192)

var transport thrift.TTransport

func RunClient(addr string, secure bool) (*casperproto.CasperServerClient, error) {

	var err error
	if secure {
		cfg := new(tls.Config)
		cfg.InsecureSkipVerify = true
		transport, err = thrift.NewTSSLSocket(addr, cfg)
	} else {
		transport, err = thrift.NewTSocket(addr)
	}
	fmt.Println("1")

	if err != nil {
		fmt.Println("Error opening socket:", err)
		return nil, err
	}

	fmt.Println("2")
	if transport == nil {
		return nil, fmt.Errorf("Error opening socket, got nil transport. Is server available?")
	}
	fmt.Println("3")
	transport, err = transportFactory.GetTransport(transport)
	if transport == nil {
		return nil, fmt.Errorf("Error from transportFactory.GetTransport(), got nil transport. Is server available?")
	}
	fmt.Println("4")
	err = transport.Open()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("5")

	return casperproto.NewCasperServerClientFactory(transport, protocolFactory), nil
}

func CloseClient() {
	transport.Close()
	return
}
