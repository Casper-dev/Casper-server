package validation

import (
	"regexp"
	"fmt"
	"math/big"
	"time"
	"gitlab.com/casperDev/Casper-SC/casper_sc"
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/casperDev/Casper-server/casper/casper_utils"
	"math/rand"
	"net"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"gitlab.com/casperDev/Casper-SC/casper"
	"gitlab.com/casperDev/Casper-server/casper/thrift"
)

type Pinger struct {
	casperclient *casper.Casper
	client       *ethclient.Client
	auth         *bind.TransactOpts
}

func (pinger *Pinger) RunPinger(ctx context.Context) {
	for {
		pinger.casperclient, pinger.client, pinger.auth, _ = Casper_SC.GetSC()

		getPingResult := func() (*types.Transaction, error) {
			return pinger.casperclient.GetPingTarget(pinger.auth, casper_utils.GetLocalAddr().ID())
		}
		//_, err = casperclient.WatchReturnString(nil, sink)

		pingResultData := Casper_SC.ValidateMineTX(getPingResult, pinger.client, pinger.auth)
		hashRegex := regexp.MustCompile("([a-zA-Z0-9]+)")
		hash := hashRegex.FindString(pingResultData)
		log.Info("waiting for event")

		log.Info("got node !!", hash, "!!")
		pinger.checkNodeByHash(ctx, hash)

		fmt.Println(time.Duration(2 +(rand.Int()%7)) * time.Minute)
		time.Sleep(time.Duration(2 +(rand.Int()%7)) * time.Minute)
		///fmt.Println(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Minute)
		///time.Sleep(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Minute)
	}
}

func (pinger *Pinger) checkNodeByHash(ctx context.Context, hash string) (err error) {
	ipRet, err := pinger.casperclient.GetIpPort(nil, hash)
	if err != nil {
		log.Error(err)
	}
	var success = false
	log.Info("got IP", ipRet)

	if hash == casper_utils.GetLocalAddr().ID() {
		log.Info("Pinging self")
		//return As of now, it's ok to ping self
	}

	ip, _, err := net.SplitHostPort(ipRet)
	if err != nil {
		log.Error(err)
	}
	if len(casper_utils.GetLocalAddr().Thrift().String()) < 7 {
		success = true ///It's not everyone's problem if current ip is wrong
	} else if len(ip) >= 7 {
		subnet := regexp.MustCompile("(\\d+\\.){2}").FindString(ip)
		currentSubnet := regexp.MustCompile("(\\d+\\.){2}").FindString(casper_utils.GetLocalAddr().Thrift().String())
		if subnet == currentSubnet && subnet != "" {
			success = true ///We won't ping nodes from our subnet
		} else {
			success = pinger.pingNode(ctx, ipRet, hash)
		}
	}

	log.Info("is", hash, "validation succeed?", success)
	resultData := Casper_SC.ValidateMineTX(
		func() (*types.Transaction, error) {
			return pinger.casperclient.SendPingResult(pinger.auth, hash, success)
		},
		pinger.client,
		pinger.auth)
	re := regexp.MustCompile("Banned!")
	log.Info("res data ", resultData)
	if re.FindString(resultData) != "" {
		log.Info("Go go replication~!")
		go startReplication(hash, pinger.casperclient)
	}
	return
}

func (pinger *Pinger) pingNode(ctx context.Context, ip string, hash string) (success bool) {
	log.Info("pinging", ip)
	timestamp := callPing(ctx, ip)
	log.Info("timestamp received", timestamp)
	success = true
	if timestamp == 0 {
		success = false
	}
	return
}

func callPing(ctx context.Context, ip string) (timestamp int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Info("Recovered when calling ping with", r)
		}
	}()
	ts, _ := thrift.RunClientClosure(ip, func(thriftClient *thrift.ThriftClient) (interface{}, error) {
		return thriftClient.Ping(ctx)
	})
	return ts.(int64)
}

func startReplication(hash string, casperclient *casper.Casper) {

	// We get number of banned provider files
	n, _ := casperclient.GetNumberOfFiles(nil, hash)

	for i := int64(0); i < n.Int64(); i++ {

		// For each file we get its hash and size
		hash, size, err := casperclient.GetFile(nil, hash, big.NewInt(i))
		if err != nil {
			log.Error(err)
		}
		log.Info("hs", hash, size)
		// Search for a new peer to store the file
		go replicateFile(hash, hash, size, casperclient)
	}
}

func replicateFile(hash string, blockedAddress string, size *big.Int, casperclient *casper.Casper) (ret string) {
	defer func() {
		if re := recover(); re != nil {
			log.Error(re)
		}
	}()

	peers, err := casperclient.GetPeers(nil, size)
	if err != nil {
		log.Error(err)
	}
	// GetPeers returns 4 ips, but we need only one of them
	ipPort, err := casperclient.GetIpPort(nil, peers.Id1)
	if err != nil {
		log.Error(err)
	}
	if ipPort != "" {
		thrift.RunClientClosure(ipPort, func(thriftClient *thrift.ThriftClient) (interface{}, error) {
			return thriftClient.SendReplicationQuery(context.TODO(), hash, blockedAddress, size.Int64())
		})
	}
	return
}
