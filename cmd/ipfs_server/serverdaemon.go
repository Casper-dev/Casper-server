package main

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"time"

	cu "gitlab.com/casperDev/Casper-server/casper/casper_utils"
	"gitlab.com/casperDev/Casper-server/casper/thrift"
	"gitlab.com/casperDev/Casper-server/commands"
	"gitlab.com/casperDev/Casper-server/repo/fsrepo"

	"gitlab.com/casperDev/Casper-SC/casper"
	"gitlab.com/casperDev/Casper-SC/casper_sc"
	"gitlab.com/casperDev/Casper-server/casper/casper_utils"

	"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"

	"github.com/ethereum/go-ethereum/core/types"
	reuse "github.com/libp2p/go-reuseport"
	"github.com/willscott/goturn/client"
	"github.com/fatih/color"
	val "gitlab.com/casperDev/Casper-server/casper/validation"
	"gitlab.com/casperDev/Casper-server/core"
)

var pingTimeout = 30 * time.Second

const (
	defaultStatusCheckInterval = 5 * time.Minute
	defaultVerificationInitiationInterval = 60 * time.Minute
)

func serveThrift(ctx context.Context, cctx *commands.Context) error {
	log.Infof("Initializing thrift server...")
	///TODO: to use a custom port we need to find way for other providers to get that port
	var thriftIP, thriftPort string
	if cctx.ConfigRoot != "" {
		if cfg, err := fsrepo.ConfigAt(cctx.ConfigRoot); err == nil && cfg != nil {
			// TODO: Find out if API can be not IP4 or thrift can use IP6
			thriftIP, _ = ma.StringCast(cfg.Addresses.API).ValueForProtocol(ma.P_IP4)
			thriftPort = cfg.Casper.ConnectionPort
		}
	} else {
		log.Error("config is not provided")
	}

	if thriftIP == "" {
		thriftIP = cu.GetLocalIP()
		thriftPort = "9090"
	}

	thriftIP = "0.0.0.0"

	// always run server here
	thriftAddr := net.JoinHostPort(thriftIP, thriftPort)
	err := thrift.RunServerDefault(thriftAddr, NewCasperServerHandler(cctx.ConfigRoot))
	if err != nil {
		log.Error("error running server:", err)
		return err
	}

	log.Infof("serving requests on %s ...", thriftAddr)
	return nil
}

func statusChecker() {
	time.Sleep(10 * time.Second) ///TODO: wait for daemon to initialize
	for ; ; {
		casper, _, _, err := Casper_SC.GetSC()
		if err != nil {
			fmt.Println(err)
		}
		isBanned, err := casper.VerifyReplication(nil, casper_utils.GetLocalAddr().ID())
		fmt.Println()
		if isBanned {
			bgGreen := color.New(color.BgRed).PrintfFunc()
			bgGreen("    Node is banned    ")
			fmt.Println()
		} else {
			bgGreen := color.New(color.BgGreen).PrintfFunc()
			bgGreen("    Node is online    ")
			fmt.Println()
		}
		time.Sleep(defaultStatusCheckInterval) /// Node online check doesn't need to be frequent; we can even change it to subscription model
	}
}

var initiatedCheck string

func verificationRunner(ctx context.Context) {
	for ;; {
		time.Sleep(defaultVerificationInitiationInterval) /// Node will run random uuid verification every defaultVerificationInitiationInterval(60 as of now) minutes
		casperClient, _, auth, _ := Casper_SC.GetSC()
		uuid := "lol"
		casperClient.NotifyVerificationTarget(auth, uuid, casper_utils.GetLocalAddr().ID())
	}
}

func verificationWatcher(ctx context.Context, node *core.IpfsNode) {
	verificationTargetWatcher := func(ctx context.Context, node *core.IpfsNode) (func(log *casper.CasperVerificationTarget)) {
		return func(log *casper.CasperVerificationTarget) {
			if _, isStored := core.UUIDInfoCache.Load(log.UUID); isStored {
				val.PerformValidation(ctx, node, log.UUID)
			}
		}
	}(ctx, node)
	casper_utils.SubscribeToVerificationTargetLogs(ctx, verificationTargetWatcher)

	verificationConsensusWatcher := func(ctx context.Context) (func(log *casper.CasperConsensusResult)) {
		return func(log *casper.CasperConsensusResult) {
			if _, isStored := core.UUIDInfoCache.Load(log.UUID); isStored {
				for _, peer := range log.Consensus {
					if string(peer[:31]) == casper_utils.GetLocalAddr().ID() {
						repairFile(ctx, log.UUID)
					}
				}
			}
		}
	}(ctx)
	casper_utils.SubscribeToVerificationConsensusLogs(ctx, verificationConsensusWatcher)
}

func repairFile(ctx context.Context, UUID string) {
	fmt.Println("Repairing file", UUID)
	/// Make file repair
}

// TODO: implement timeout
const connectStunTimeout = 2 * time.Minute
const connectStunKeepAlive = time.Hour

func connectStun(laddr, raddr string) (ma.Multiaddr, error) {
	var d reuse.Dialer
	if laddr != "" {
		netladdr, err := reuse.ResolveAddr("tcp", laddr)
		if err != nil {
			return nil, err
		}
		d.D.LocalAddr = netladdr
	}
	d.D.Timeout = connectStunTimeout
	d.D.KeepAlive = connectStunKeepAlive

	conn, err := d.Dial("tcp", raddr)
	if err != nil {
		log.Error("Could not connect to STUN server:", err)
		return nil, err
	}

	// We do not close connection intentionally, so that
	// our external (outer) address remains the same
	//defer conn.Close()
	c := client.StunClient{Conn: conn}
	address, err := c.Bind()
	if err != nil {
		log.Error("Failed to bind:", err)
		return nil, err
	}

	// StunClient.Bind() provides its own 'net.Addr' implementation
	// which cannot be used as argument to FromNetAddr.
	// So we construct net.TCPAddr here explicitly.
	tcp, err := net.ResolveTCPAddr(address.Network(), address.String())
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return manet.FromNetAddr(tcp)
}

func runPinger(ctx context.Context) {
	///beforeTimeout := time.Duration(60 - time.Now().Minute()) * time.Second
	beforeTimeout := time.Duration(30) * time.Second
	fmt.Printf("Waiting for %s\n", beforeTimeout)
	time.Sleep(beforeTimeout)

	for {
		casperclient, client, auth, _ := Casper_SC.GetSC()
		fmt.Println("auth nonce", auth.Nonce)
		nonce, _ := client.PendingNonceAt(ctx, auth.From)
		fmt.Println("pending nonce", nonce)
		//sink := make(chan *casper.CasperReturnString)
		//_, err = casperclient.WatchReturnString(nil, sink)
		nonce, _ = client.PendingNonceAt(ctx, auth.From)
		fmt.Println("pending nonce", nonce)

		rer := regexp.MustCompile("(/ip4/[/\\d\\w.]+)")
		ipRet := strings.TrimSpace(Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
			return casperclient.GetPingTarget(auth, cu.GetLocalAddr().String())
		}, client, auth))
		fmt.Println("waiting for event")
		//retString := <-sink
		//fmt.Println("event ret ", retString.Val)

		re := regexp.MustCompile("/.+?/(.+?)/")
		fmt.Println("got ip !!", ipRet, "!!")
		ipRet = rer.FindString(ipRet)
		fmt.Println("got ip !!", ipRet, "!!")
		fmt.Println(len(ipRet))
		ips := re.FindStringSubmatch(ipRet)
		if len(ips) >= 1 {
			log.Debugf("Ping: %s", ips[0])
			if timestamp, err := callPing(ctx, net.JoinHostPort(ips[0], "9090")); err == nil {
				log.Debugf("Ping timestamp: %d", timestamp)
				success := timestamp >= time.Now().Unix()-int64(pingTimeout.Seconds())
				nonce, _ = client.PendingNonceAt(ctx, auth.From)
				log.Debugf("Pending nonce", nonce)
				resultData := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
					return casperclient.SendPingResult(auth, ipRet, success)
				}, client, auth)

				///TODO: change to working regex
				re := regexp.MustCompile("Banned!")
				fmt.Println("res data ", resultData)
				if re.FindString(resultData) != "" {
					fmt.Println("Go go replication~!")
					go startReplication(ctx, ipRet, casperclient)
				}
			}
		}

		///pingTimeout := time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Second
		pingWait := time.Duration(10+rand.Intn(20)) * time.Second
		fmt.Printf("Sleeping for %s\n", pingWait)
		time.Sleep(pingTimeout)
	}
}

func callPing(ctx context.Context, ip string) (timestamp int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	ts, err := thrift.RunClientClosure(ip, func(client *thrift.ThriftClient) (interface{}, error) {
		return client.Ping(ctx)
	})
	if err != nil {
		return 0, err
	}
	return ts.(int64), nil
}

func startReplication(ctx context.Context, ip string, casperclient *casper.Casper) {
	// We get number of banned provider files
	n, err := casperclient.GetNumberOfFiles(nil, ip)
	if err != nil {
		log.Error(err)
		return
	}

	for i := int64(0); i < n.Int64(); i++ {
		// For each file we get its hash and size
		hash, size, err := casperclient.GetFile(nil, ip, big.NewInt(i))
		if err != nil {
			log.Error(err)
			continue
		}

		log.Debug("hs", hash, size)

		// Search for a new peer to store the file
		go replicateFile(ctx, hash, size, casperclient)
	}
}

func replicateFile(ctx context.Context, hash string, size *big.Int, casperclient *casper.Casper) (ret string) {
	defer func() {
		if re := recover(); re != nil {
			fmt.Println(re)
		}
	}()

	tx, err := casperclient.GetPeers(nil, size)
	if err != nil {
		fmt.Println(err)
	}
	// GetPeers returns 4 ips, but we need only one of them
	filteredIP, err := casperclient.GetIpPort(nil, tx.Id1)
	if err != nil {
		fmt.Println(err)
	}

	if filteredIP != "" {
		ip := net.JoinHostPort(filteredIP, "9090")
		thrift.RunClientClosure(ip, func(client *thrift.ThriftClient) (interface{}, error) {
			return client.SendReplicationQuery(ctx, hash, ip, size.Int64())
		})
	}
	return
}
