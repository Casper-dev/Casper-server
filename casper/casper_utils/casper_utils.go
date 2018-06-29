package casper_utils

import (
	"context"
	"expvar"
	"fmt"
	"net"
	"regexp"

	"github.com/Casper-dev/Casper-server/casper/sc"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/repo/config"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"

	"github.com/fatih/color"
)

var log = logging.Logger("csp/utils")

type ExternalAddr struct {
	IPFSAddr   ipfsaddr.IPFSAddr
	ThriftAddr *net.TCPAddr
}

var localNode *ExternalAddr

func (a *ExternalAddr) Thrift() *net.TCPAddr {
	return a.ThriftAddr
}
func (a *ExternalAddr) IPFS() ipfsaddr.IPFSAddr {
	return a.IPFSAddr
}
func (a *ExternalAddr) String() string {
	return a.IPFSAddr.String()[:]
}

func GetLocalAddr() *ExternalAddr {
	return localNode
}

func (a *ExternalAddr) NodeHash() string {
	return a.IPFSAddr.ID().Pretty()
}

var ErrInvalidLocalAddr = fmt.Errorf("cannot determine IP for SC registration")

func initDebug() {
	var getInfo expvar.Func = func() interface{} {
		if localNode == nil {
			return nil
		}
		return map[string]string{
			"thrift": localNode.ThriftAddr.String(),
			"ipfs":   localNode.IPFS().String(),
		}
	}
	expvar.Publish("localnode", getInfo)
}

func RegisterSC(ctx context.Context, node *core.IpfsNode, cfg *config.Config, addresses ...string) error {
	initDebug()

	if len(addresses) > 0 {
		addr := ma.StringCast(addresses[0])
		if naddr, err := manet.ToNetAddr(addr); err == nil {
			fmt.Println("Thrift external IPs were provided:", addresses)
			addrS := fmt.Sprintf("%s/ipfs/%s", addr, node.Identity.Pretty())
			if id, err := ipfsaddr.ParseString(addrS); err == nil {
				localNode = &ExternalAddr{id, naddr.(*net.TCPAddr)}
			} else {
				log.Error(err)
			}
		}
	}

	if localNode == nil {
		fmt.Println("No external IPs were provided")
		var addr ma.Multiaddr
		var ip string
		for _, str := range cfg.Addresses.Swarm {
			var err error
			addr = ma.StringCast(str)
			if ip, err = addr.ValueForProtocol(ma.P_IP4); err == nil {
				break
			}
		}
		if addr == nil {
			log.Error("nil address")
			return ErrInvalidLocalAddr
		}
		iaddr := ma.StringCast("/ipfs/" + node.Identity.Pretty())
		id, err := ipfsaddr.ParseMultiaddr(addr.Encapsulate(iaddr))
		if err != nil {
			log.Error("failed to parse multiaddr:", err)
			return ErrInvalidLocalAddr
		}
		localNode = &ExternalAddr{id, &net.TCPAddr{IP: net.ParseIP(ip), Port: 9090}}
	}

	fmt.Println("Full node address:", localNode.String())

	settings, ok := cfg.Casper.Blockchain[cfg.Casper.UsedChain]
	if !ok {
		log.Warning("no settings for connection to blockchain are specified, using default")
		settings = nil
	}

	c, err := sc.GetContractByName(ctx, cfg.Casper.UsedChain, settings)
	if err != nil {
		return fmt.Errorf("cant initialize SC: %v", err)
	}

	///TODO: debug only: remove as soon as we deploy
	geoloc := GetGeoloc()
	fmt.Println(geoloc)

	c.AddToken(13370000000)

	taddr := localNode.Thrift()
	connectionString := regexp.MustCompile("ip4/.+/tcp").ReplaceAllString(cfg.Addresses.Swarm[0], "ip4/"+taddr.IP.String()+"/tcp")
	fmt.Println("Conn string", connectionString)
	nodeID := localNode.IPFSAddr.ID().Pretty()
	err = c.RegisterProvider(nodeID, cfg.Casper.TelegramAddress, connectionString, taddr.String(), 13370000000)
	if err != nil { // the provider is already registered
		err = c.SetRPCAddr(nodeID, taddr.String())
		if err != nil {
			if isBanned, _ := c.VerifyReplication(localNode.NodeHash()); isBanned {
				color.New(color.BgRed).Print("    Node is banned    ")
				fmt.Println()
			}
			return fmt.Errorf("cant update IP in SC: %v", err)
		}
	} else {
		err = c.SetRPCAddr(nodeID, taddr.String())
		if err != nil {
			log.Error(err)
		}
		err = c.SetOriginCode(nodeID, geoloc)
		if err != nil {
			log.Error(err)
		}
	}
	log.Info("Node registered")
	return nil
}

func StringGet32Lower(inputString string) string {
	return inputString[len(inputString)-31:]
}

func GetPeersMultiaddrs(hash string) ([]ma.Multiaddr, error) {
	c, _ := sc.GetContract()
	peers, err := c.ShowStoringPeers(hash)
	if err != nil {
		return nil, err
	}
	return getMultiaddrsByPeers(peers)
}

func GetPeersMultiaddrsBySize(size int64, count int) (ret []ma.Multiaddr, err error) {
	c, _ := sc.GetContract()
	peers, err := c.GetPeers(size, count)
	if err != nil {
		log.Error(err)
	}
	return getMultiaddrsByPeers(peers)
}

func getMultiaddrsByPeers(peers []string) (ret []ma.Multiaddr, err error) {
	c, _ := sc.GetContract()
	fmt.Println(peers)
	for _, peer := range peers {
		if peer == "" {
			log.Error("empty peer ID")
			continue
		}
		func(peer string) {
			ipPort, err := c.GetAPIAddr(peer)
			if err != nil {
				log.Error(err)
				return
			}

			mastr := fmt.Sprintf("%s/ipfs/%s", ipPort, peer)
			if maddr, err := ma.NewMultiaddr(mastr); err == nil {
				ret = append(ret, maddr)
			} else {
				log.Error(err)
			}

		}(peer)
	}
	return ret, err
}

func GetPeersMultiaddrsByHash(hash string) (ret []ma.Multiaddr, err error) {
	return GetPeersMultiaddrs(hash)
}

func GetIpPortsByHash(hash string) (ret []string) {
	// FIXME
	c, _ := sc.GetContract()
	peers, err := c.ShowStoringPeers(hash)
	if err != nil {
		log.Error(err)
		return
	}
	for _, peer := range peers {
		ipPort, err := c.GetRPCAddr(peer)
		if err != nil {
			fmt.Println(err)
			continue
		}
		ret = append(ret, ipPort)
	}
	return
}
