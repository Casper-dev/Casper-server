package casper_utils

import (
	"fmt"

	"github.com/Casper-dev/Casper-server/core"

	"time"
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"git.apache.org/thrift.git/lib/go/thrift"
	"crypto/tls"

	"github.com/Casper-dev/Casper-SC/casper_sc"
	"github.com/Casper-dev/Casper-server/casper/casper_thrift/casperproto"
)

var fullNodeID string

func RegisterSC(node *core.IpfsNode) {
	fmt.Println(node.Identity.Pretty())
	casper, client, auth := Casper_SC.GetSC()
	ip := Casper_SC.GetIPv4()

	fullNodeID = "/ip4/" + ip + "/tcp/4001/ipfs/" + node.Identity.Pretty()
	fmt.Println(fullNodeID)

	///TODO: debug only: remove as soon as we deploy
	txAdd, err := casper.AddToken(auth, big.NewInt(int64(13370000000)))
	ValidateMineTX(txAdd, client, err)
	fmt.Println("registering")

	tx, err := casper.RegisterProvider(auth, fullNodeID, big.NewInt(int64(13370000000)))
	if err != nil && err.Error() == "exceeds block gas limit" {
		auth.GasLimit = uint64(500000)
		tx, err = casper.RegisterProvider(auth, fullNodeID, big.NewInt(int64(13370000000)))
		fmt.Println(err)
	}

	ValidateMineTX(tx, client, err)

	return
}

func ValidateMineTX(tx *types.Transaction, client *ethclient.Client, err error) (data string) {
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Pending TX: 0x%x\n", tx.Hash())
		data = MineTX(tx, client)
	}
	return
}

func MineTX(tx *types.Transaction, client *ethclient.Client) (data string) {
	fmt.Printf("Gas %d\nGas price %d", tx.Gas(), tx.GasPrice())
	for ; ; {
		rxt, pending, err := client.TransactionByHash(context.Background(), tx.Hash())
		if err != nil {
			println(err)
		} else {
			println("Waiting for TX to mine")
		}
		if (!pending) {
			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			fmt.Println(err)
			if err == nil {
				println(rxt.String())
				println("receipt", receipt.Status, receipt.String())
				if len(receipt.Logs) > 0 {
					data = string(receipt.Logs[0].Data)
					fmt.Println(data)
				}
			}

			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("Tx succesfully mined")
	return
}

func GetCasperNodeID() string {
	return fullNodeID
}

var protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()

var transportFactory = thrift.NewTBufferedTransportFactory(8192)

func RunClient(addr string, secure bool) (*casperproto.CasperServerClient, error) {
	var transport thrift.TTransport
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
	defer transport.Close()

	return casperproto.NewCasperServerClientFactory(transport, protocolFactory), nil
}
