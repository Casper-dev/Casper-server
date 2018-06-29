package main

import (
	"context"
	"fmt"
	"net"
	"time"

	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	"github.com/Casper-dev/Casper-server/casper/sc"
	"github.com/Casper-dev/Casper-server/casper/thrift"
	"github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/repo/fsrepo"

	"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"

	"github.com/fatih/color"
	reuse "github.com/libp2p/go-reuseport"
	"github.com/willscott/goturn/client"
)

var pingTimeout = 30 * time.Second

const (
	defaultStatusCheckInterval            = 5 * time.Minute
	defaultStatusCheckTimeout             = 1 * time.Minute
	defaultVerificationInitiationInterval = 60 * time.Minute
	defaultVerificationInitiationTimeout  = 1 * time.Minute
)

func serveThrift(ctx context.Context, cctx *commands.Context) error {
	log.Infof("Initializing thrift server...")

	thriftIP := "0.0.0.0"
	thriftPort := "9090"
	if cctx.ConfigRoot != "" {
		if cfg, err := fsrepo.ConfigAt(cctx.ConfigRoot); err == nil && cfg != nil {
			// TODO: Find out if API can be not IP4 or thrift can use IP6
			//thriftIP, _ = ma.StringCast(cfg.Addresses.API).ValueForProtocol(ma.P_IP4)
			thriftPort = cfg.Casper.ConnectionPort
		}
	} else {
		log.Error("config is not provided")
	}

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

func statusChecker(ctx context.Context) {
	time.Sleep(10 * time.Second) ///TODO: wait for daemon to initialize

	// This is in separate function, because first tick in time.Ticker
	// does not occur instantly.
	checkStatus := func(ctx context.Context) {
		c, err := sc.GetContract()
		if err != nil {
			log.Errorf("error while getting SC: %v", err)
			return
		}

		// TODO implement timeout
		//tctx, cancel := context.WithTimeout(ctx, defaultStatusCheckTimeout)
		isBanned, err := c.VerifyReplication(cu.GetLocalAddr().NodeHash())
		//cancel()
		if err != nil {
			log.Error(err)
		}

		fmt.Println()
		if isBanned {
			color.New(color.BgRed).Print("    Node is banned    ")
		} else {
			color.New(color.BgGreen).Print("    Node is online    ")
		}
		fmt.Println()
	}

	/// Node online check doesn't need to be frequent; we can even change it to subscription model
	ticker := time.NewTicker(defaultStatusCheckInterval)
	checkStatus(ctx)
	for {
		select {
		case <-ticker.C:
			checkStatus(ctx)
		case <-ctx.Done():
			log.Error(ctx.Err())
			return
		}
	}
}

var initiatedCheck string

func verificationRunner(ctx context.Context) {
	/// Node will run random uuid verification every defaultVerificationInitiationInterval(60 as of now) minutes
	ticker := time.NewTicker(defaultVerificationInitiationInterval)
	for {
		select {
		case <-ticker.C:
			c, err := sc.GetContract()
			if err != nil {
				log.Errorf("error while getting SC: %v", err)
				continue
			}
			//tctx, cancel := context.WithTimeout(ctx, defaultVerificationInitiationTimeout)
			//auth.Context = tctx
			// FIXME UUID
			err = c.NotifyVerificationTarget("123", cu.GetLocalAddr().NodeHash())
			//cancel()
			if err != nil {
				/// Non-critical error; logging to info/debug is ok
				log.Info(err)
				continue
			}
		case <-ctx.Done():
			log.Error(ctx.Err())
			return
		}
	}
}

// TODO move to interface somehow
func verificationWatcher(ctx context.Context, node *core.IpfsNode) {
	// FIXME when we come up with new algorithm without events subscribing
	return
	//c, _ := sc.GetContract()
	//c.SubscribeVerificationTarget(ctx, func(uuid string, id string) {
	//	if _, ok := core.UUIDInfoCache.Load(uuid); ok {
	//		val.PerformValidation(ctx, node, uuid)
	//	}
	//})

	//c.SubscribeConsensusResult(ctx, func(uuid string, consensus [4][32]byte) {
	//	if _, ok := core.UUIDInfoCache.Load(uuid); ok {
	//		for _, peer := range consensus {
	//			// FIXME
	//			if string(peer[:31]) == cu.GetLocalAddr().NodeHash() {
	//				repairFile(ctx, uuid)
	//			}
	//		}
	//	}
	//})
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
