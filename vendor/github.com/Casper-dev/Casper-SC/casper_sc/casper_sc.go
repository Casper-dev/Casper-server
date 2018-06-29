package Casper_SC

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/Casper-dev/Casper-SC/casper"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

var log = logging.Logger("SC")

// with no 0x
const ContractAddress = "d3a9e2d2b34F87302569f5Bf2aBed5969A2A5925"

// with no 0x
const PrivateKey = "674393e0fb1cba8a71be3f1261e7171effb998bc5047ae0eee8b0e49e556e293"
const Gateway = "http://94.130.182.144:8775"

func GetIPv4() (ip string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Error(err)
		return ""
	}
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil && !v.IP.IsLoopback() {
					ip = v.IP.String()
					log.Infof("got ipv4: %s, %s\n", ip, addr.Network())
				}
			}
		}
	}
	return
}

type InitOpts = struct {
	ContractAddress string
	Gateway         string
	PrivateKey      string
}

var casperClient *casper.Casper
var client *ethclient.Client
var auth *bind.TransactOpts
var mu sync.Mutex

func Initialized() bool {
	if casperClient == nil {
		return false
	}
	_, _, _, _, err := casperClient.GetPeers(nil, big.NewInt(1), big.NewInt(1))
	return err == nil
}

func InitSC(ctx context.Context, opts *InitOpts) (*casper.Casper, *ethclient.Client, *bind.TransactOpts, error) {
	if opts == nil {
		panic("empty options while initializing SC")
	}

	client, err := ethclient.DialContext(ctx, opts.Gateway)
	if err != nil {
		return nil, nil, nil, err
	}

	addr := ContractAddress
	if opts.ContractAddress != "" {
		addr = opts.ContractAddress
	}
	contractAddress := common.HexToAddress(addr)
	caspersc, err := casper.NewCasper(contractAddress, client)
	if err != nil {
		return nil, nil, nil, err
	}
	key, err := crypto.HexToECDSA(opts.PrivateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	auth := bind.NewKeyedTransactor(key)
	// TODO add comment about constant
	auth.GasLimit = uint64(1500000)
	return caspersc, client, auth, nil
}

func GetSC() (*casper.Casper, *ethclient.Client, *bind.TransactOpts, error) {
	mu.Lock()
	defer mu.Unlock()
	if casperClient != nil {
		_, _, _, _, err := casperClient.GetPeers(nil, big.NewInt(1), big.NewInt(1))
		if err != nil {
			return nil, nil, nil, err
		}
		return casperClient, client, auth, err
	}
	return nil, nil, nil, errors.New("not initialized")
}

func GetWebsocketClient() (*ethclient.Client, error) {
	client, err := ethclient.Dial("ws://94.130.182.144:8776")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return client, nil
}

func GetWebsockSC() (*casper.Casper, error) {
	log.Info("getting client")
	client, err := GetWebsocketClient()
	if err != nil {
		return nil, err
	}
	log.Info("got ws client")
	casperSClient, err := casper.NewCasper(common.HexToAddress(ContractAddress), client)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return casperSClient, nil
}

func ValidateMineTX(txGet func() (tx *types.Transaction, err error), client *ethclient.Client) (string, error) {
	tx, err := txGet()
	for err != nil {
		/// Non-critical error; logging to info/debug is ok
		log.Info(err)
		time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(3000)))
		tx, err = txGet()
	}
	log.Infof("Pending TX: 0x%x\n", tx.Hash())
	return MineTX(tx, client)
}

func MineTX(tx *types.Transaction, client *ethclient.Client) (data string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("recovered in MineTX():", r)
		}
	}()

	log.Infof("Gas %d; Gas price %d", tx.Gas(), tx.GasPrice())
	for {
		rxt, pending, err := client.TransactionByHash(context.Background(), tx.Hash())
		if err != nil {
			return "", err
		}
		log.Info("Waiting for TX to mine")
		if !pending {
			log.Info("Waiting a second for the receipt")
			time.Sleep(1 * time.Second)
			log.Info("getting receipt")
			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				return "", err
			}
			log.Infof("TX: hash=%s, nonce=%d, gas=%d ", rxt.Hash().String(), rxt.Nonce, rxt.Gas)
			log.Infof("RECEIPT: %d %s %v", receipt.Status, receipt.TxHash.String(), receipt.Logs)
			if receipt.Status == types.ReceiptStatusFailed {
				//return "", errors.New("transaction receipt status is 0")
			}
			if len(receipt.Logs) > 0 {
				for _, receiptLog := range receipt.Logs {
					data += string(receiptLog.Data)
				}
				log.Infof("data: %s", data)
			}
			break
		}
		time.Sleep(2500 * time.Millisecond)

	}
	log.Info("Tx succesfully mined")
	return data, nil
}

func SubscribeToReplicationLogs(ctx context.Context, callback func(target *casper.CasperVerificationTarget)) error {
	casperSClient, err := GetWebsockSC()
	if err != nil {
		return err
	}

	replicationTargetChan := make(chan *casper.CasperVerificationTarget, 2)
	sub, err := casperSClient.WatchVerificationTarget(nil, replicationTargetChan)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Looping")
	errChan := sub.Err()
	for {
		select {
		case err := <-errChan:
			log.Error("Logs subscription error", err)
			return err
		case logEntry := <-replicationTargetChan:
			callback(logEntry)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func SubscribeToConsensusLogs(ctx context.Context, callback func(result *casper.CasperConsensusResult)) error {
	casperSClient, err := GetWebsockSC()
	if err != nil {
		return err
	}

	consResultChan := make(chan *casper.CasperConsensusResult, 2)
	sub, err := casperSClient.WatchConsensusResult(nil, consResultChan)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Looping")
	errChan := sub.Err()
	for {
		select {
		case err := <-errChan:
			log.Error("Logs subscription error", err)
			return err
		case logEntry := <-consResultChan:
			callback(logEntry)
		}
	}
}

func SubscribeToProvidersCheckLogs(ctx context.Context, callback func(result *casper.CasperProviderCheckEvent)) error {
	casperSClient, err := GetWebsockSC()
	if err != nil {
		return err
	}

	consResultChan := make(chan *casper.CasperProviderCheckEvent, 2)
	sub, err := casperSClient.WatchProviderCheckEvent(nil, consResultChan)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Looping")
	errChan := sub.Err()
	for {
		select {
		case err := <-errChan:
			log.Error("Logs subscription error", err)
			return err
		case logEntry := <-consResultChan:
			callback(logEntry)
		}
	}
}
