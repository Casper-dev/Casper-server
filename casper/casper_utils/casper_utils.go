package casper_utils

import (
	"context"

	"fmt"
	"math/big"
	"net"
	"gitlab.com/casperDev/Casper-SC/casper"
	"gitlab.com/casperDev/Casper-SC/casper_sc"
	"gitlab.com/casperDev/Casper-server/core"
	"gitlab.com/casperDev/Casper-server/repo/config"

	"github.com/ethereum/go-ethereum/core/types"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"
	"regexp"
)

var log = logging.Logger("csp/utils")

type ExternalAddr struct {
	IPFSAddr   ipfsaddr.IPFSAddr
	ThriftAddr net.Addr
}

var localNode *ExternalAddr

func (a *ExternalAddr) Thrift() net.Addr {
	return a.ThriftAddr
}
func (a *ExternalAddr) IPFS() ipfsaddr.IPFSAddr {
	return a.IPFSAddr
}
func (a *ExternalAddr) String() string {
	a.IPFSAddr.String()

	return a.IPFSAddr.String()[:]
}

func GetLocalAddr() *ExternalAddr {
	return localNode
}

func (a *ExternalAddr) ID() string {
	return StringGet32Lower(a.IPFSAddr.ID().Pretty())
}

var ErrInvalidLocalAddr = fmt.Errorf("cannot determine IP for SC registration")

func RegisterSC(node *core.IpfsNode, cfg *config.Config, addresses ...string) error {
	if len(addresses) > 0 {
		if addr, err := manet.ToNetAddr(ma.StringCast(addresses[0])); err == nil {

			fmt.Println("Thrift external IPs were provided:", addresses, addr)
			id, err := ipfsaddr.ParseString(fmt.Sprintf("%s/ipfs/%s", addresses[0], node.Identity.Pretty()))
			if err == nil {
				fmt.Println("Making local thrift")
/*
			fmt.Println("Thrift external IPs were provided:", addresses)
			addrS := fmt.Sprintf("%s/ipfs/%s", addr, node.Identity.Pretty())
			if id, err := ipfsaddr.ParseString(addrS); err == nil {
*/
				localNode = &ExternalAddr{id, addr}
			} else {
				fmt.Println("err while parsing", err)
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

	//nodeID = StringGet32Lower(node.Identity.Pretty())
	casperClient, client, auth, _ := Casper_SC.GetSC()

	///TODO: debug only: remove as soon as we deploy
	Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return casperClient.AddToken(auth, big.NewInt(int64(13370000000)))
	}, client, auth)

	udpIp, _, _ := net.SplitHostPort(localNode.Thrift().String())
	connectionString := regexp.MustCompile("ip4/.+/tcp").ReplaceAllString(cfg.Addresses.Swarm[0], "ip4/"+udpIp+"/tcp")
	fmt.Println("Conn string", connectionString)
	var telegramAddress [32]byte
	copy(telegramAddress[:], []byte(cfg.Casper.TelegramAddress)[:31])
	Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return casperClient.RegisterProvider(auth, localNode.IPFSAddr.ID().Pretty(), telegramAddress, connectionString, localNode.Thrift().String(), StringGet32Lower(localNode.String()), big.NewInt(int64(13370000000)))
	}, client, auth)
	Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return casperClient.UpdateIpPort(auth, StringGet32Lower(localNode.String()), localNode.Thrift().String())
	}, client, auth)

	return nil
}

func GetNodeConnectionString() string {
	return localNode.String()
}

func StringGet32Lower(inputString string) string {
	return inputString[len(inputString)-31:]
}

func GetPeersMultiaddrs(hash string) (ret []ma.Multiaddr, err error) {
	caspersclient, _, _, _ := Casper_SC.GetSC()
	bytePeers, err := caspersclient.ShowStoringPeers(nil, hash)
	if err != nil {
		fmt.Println(err)
	}
	var peers []string
	for i := 0; i < len(bytePeers); i++ {
		peers = append(peers, string(bytePeers[i][:31]))
	}
	getMultiaddrsByPeers(peers)
	return ret, err
}

func GetPeersMultiaddrsBySize(size int64) (ret []ma.Multiaddr, err error) {
	caspersclient, _, _, _ := Casper_SC.GetSC()
	peersStruct, err := caspersclient.GetPeers(nil, big.NewInt(size))
	if err != nil {
		fmt.Println(err)
	}
	peers := []string{peersStruct.Id1, peersStruct.Id2, peersStruct.Id3, peersStruct.Id4}
	return getMultiaddrsByPeers(peers)
}

func getMultiaddrsByPeers(peers []string) (ret []ma.Multiaddr, err error) {
	caspersclient, _, _, _ := Casper_SC.GetSC()
	fmt.Println(peers)
	for _, peer := range peers {
		func(peer string) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Error with peer", peer, r)
				}
			}()
			ipPort, err := caspersclient.GetUDPIpPort(nil, peer)
			hash, err := caspersclient.GetNodeHash(nil, peer)
			fmt.Println(ipPort + "/ipfs/" + hash)
			if err != nil {
				fmt.Println(err)
				return
			}
			if maddr, err := ma.NewMultiaddr(ipPort + "/ipfs/" + hash); err == nil {
				ret = append(ret, maddr)
			} else {
				fmt.Println(err)
			}
		}(peer)
	}
	return ret, err
}

func GetPeersMultiaddrsByHash(hash string) (ret []ma.Multiaddr, err error) {
	return GetPeersMultiaddrs(hash)
}

func GetPeersForUpload(size int64) (ret []string) {
	//return []string{"10.10.10.1"}
	client, _, _, _ := Casper_SC.GetSC()
	tx_get, _ := client.GetPeers(nil, big.NewInt(size))
	peers := []string{tx_get.Id1, tx_get.Id2, tx_get.Id3, tx_get.Id4}

	for _, peer := range peers {
		if filteredIP, err := client.GetIpPort(nil, peer); err == nil && filteredIP != "" {
			ret = append(ret, filteredIP)
		} else {
			fmt.Println(err)
		}
	}
	return
}

func GetIpPortsByHash(hash string) (ret []string) {
	// FIXME
	caspersclient, _, _, _ := Casper_SC.GetSC()
	peers, err := caspersclient.ShowStoringPeers(nil, hash)
	if err != nil {
		fmt.Println(err)
	}
	for _, peer := range peers {
		ipPort, err := caspersclient.GetIpPort(nil, string(peer[:31]))
		if err != nil {
			fmt.Println(err)
			continue
		}
		ret = append(ret, ipPort)
	}
	return
}

func SubscribeToVerificationTargetLogs(ctx context.Context, logCallback func(log *casper.CasperVerificationTarget)) {
	Casper_SC.SubscribeToReplicationLogs(ctx, logCallback)
}

func SubscribeToVerificationConsensusLogs(ctx context.Context, logCallback func(log *casper.CasperConsensusResult)) {
	Casper_SC.SubscribeToConsensusLogs(ctx, logCallback)
}
