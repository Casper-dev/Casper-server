package validation

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	"github.com/Casper-dev/Casper-server/casper/sc"
	scint "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"
	"github.com/Casper-dev/Casper-server/casper/thrift"

	"github.com/Casper-dev/Casper-thrift/casperproto"
)

type Pinger struct {
	sc               scint.CasperSC
	lastOverseerTime int64
}

const (
	DefaultPingInterval    = 2
	OverseerPingInterval   = 2
	OverseerActiveTime     = 3600
	replicateAttemptsCount = 4
)

func (pinger *Pinger) RunPinger(ctx context.Context) {
	/*	For now we'll use parsing by hand; filterers don't work at all with parity it seems
		providerEvent := make(chan string, 1)
		go cu.SubscribeToProviderCheckLogs(ctx, func(log *casper.CasperProviderCheckEvent) {
			fmt.Println("received log", log.Id)
			providerEvent <- log.Id
		})*/
	for {
		pingInterval := DefaultPingInterval /// resulting interval is [pingInterval; 2*pingInterval)
		if pinger.lastOverseerTime+OverseerActiveTime > time.Now().Unix() {
			pingInterval = OverseerPingInterval
		}

		sleepTime := time.Duration(pingInterval+rand.Intn(pingInterval)) * time.Minute
		fmt.Println(sleepTime)
		time.Sleep(sleepTime)

		pinger.sc, _ = sc.GetContract()
		hash, isOverseer, err := pinger.sc.GetPingTarget(cu.GetLocalAddr().NodeHash())
		if err != nil {
			log.Error(err)
			continue
		}

		log.Infof("got node: %s", hash)
		if isOverseer {
			log.Info("Node was made an overseer", time.Now())
			pinger.lastOverseerTime = time.Now().Unix() + OverseerActiveTime
		}

		if err = pinger.checkNodeByHash(ctx, hash); err != nil {
			log.Error(err)
		}
	}
}

func (pinger *Pinger) checkNodeByHash(ctx context.Context, hash string) (err error) {
	if hash == cu.GetLocalAddr().NodeHash() {
		log.Info("pinging self")
		return nil
	}

	ipRet, err := pinger.sc.GetRPCAddr(hash)
	if err != nil {
		return err
	}
	log.Infof("got ip: %s", ipRet)
	host, _, err := net.SplitHostPort(ipRet)
	if err != nil {
		log.Error(err)
	}

	success := false
	if taddr := cu.GetLocalAddr().Thrift(); taddr == nil {
		// TODO we probably should panic here
		success = true ///It's not everyone's problem if current ip is wrong
	} else if ip := net.ParseIP(host); ip != nil {
		// FIXME store subnet do not ping if either of IPs is reserved
		mask := net.IPv4Mask(0xff, 0xff, 0xff, 0)
		success = ip.Mask(mask).Equal(taddr.IP.Mask(mask)) || pinger.pingNode(ctx, ipRet, hash)
	}

	log.Infof("node '%s' validation succeeded: %t", hash, success)
	isBanned, err := pinger.sc.SendPingResult(hash, success)
	if err != nil {
		log.Error("error while validating TX:", err)
	} else if isBanned {
		log.Info("Go go replication~!")
		go pinger.startReplication(ctx, hash)
	}
	return
}

func (pinger *Pinger) pingNode(ctx context.Context, ip string, hash string) bool {
	log.Infof("pinging: %s", ip)
	timestamp, nodeID, err := callPing(ctx, ip)
	if err != nil {
		log.Error("error during ping:", err)
		return false
	}
	log.Infof("timestamp received: %d", timestamp)
	log.Infof("node id received  : %s", nodeID)
	return timestamp != 0 && hash == nodeID
}

func callPing(ctx context.Context, ip string) (int64, string, error) {
	r, err := thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.Ping(ctx)
	})
	if err != nil {
		return 0, "", err
	}
	p := r.(*casperproto.PingResult_)
	return p.Timestamp, p.ID, nil
}

func (pinger *Pinger) startReplication(ctx context.Context, hash string) (err error) {
	// We get number of banned provider files
	n, err := pinger.sc.GetNumberOfFiles(hash)
	if err != nil {
		log.Error("error while trying to receive number of files:", err)
		return err
	}
	log.Debugf("number of files on node '%s': %d", hash, n)
	for i := int64(0); i < n; i++ {
		// For each file we get its hash and size
		id, size, err := pinger.sc.GetFile(hash, i)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Infof("got file '%s' of size %d", id, size)
		// Search for a new peer to store the file
		go pinger.replicateFile(ctx, id, hash, size)
	}
	return nil
}

func (pinger *Pinger) replicateFile(ctx context.Context, hash string, blockedAddress string, size int64) (ret string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("recovered in replicateFile():", r)
		}
	}()

	peers, err := pinger.sc.GetPeers(size, replicateAttemptsCount)
	if err != nil {
		log.Error(err)
		return
	}

	success := false
	for _, peer := range peers {
		ipPort, err := pinger.sc.GetRPCAddr(peer)
		if err != nil {
			log.Error(err)
			continue
		}

		_, err = thrift.RunClientClosure(ipPort, func(thriftClient *thrift.ThriftClient) (interface{}, error) {
			return thriftClient.SendReplicationQuery(ctx, hash, blockedAddress, size)
		})

		if success = err == nil; success {
			break
		}
		log.Error(err)
	}

	if !success {
		log.Errorf("cannot replicate file '%s'", hash)
	}

	return
}
