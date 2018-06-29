package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	"github.com/Casper-dev/Casper-server/casper/proxy"
	"github.com/Casper-dev/Casper-server/casper/sc"
	"github.com/Casper-dev/Casper-server/casper/thrift"
	uid "github.com/Casper-dev/Casper-server/casper/uuid"
	val "github.com/Casper-dev/Casper-server/casper/validation"
	oldCmds "github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/core/commands"
	"github.com/Casper-dev/Casper-server/core/corehttp"
	"github.com/Casper-dev/Casper-server/exchange/bitswap/decision"
	"github.com/Casper-dev/Casper-server/repo/fsrepo"

	"github.com/Casper-dev/Casper-thrift/casperproto"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	"gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"
)

func CasperThriftOption(cctx oldCmds.Context) corehttp.ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		cmdHandler := thrift.NewHandler(NewCasperServerHandler(cctx.ConfigRoot))
		mux.Handle(thrift.CasperThriftApi+"/", cmdHandler)
		return mux, nil
	}
}

type CasperServerHandler struct {
	configRoot string
}

func NewCasperServerHandler(configRoot string) *CasperServerHandler {
	return &CasperServerHandler{configRoot: configRoot}
}

func (_ *CasperServerHandler) NodeID() string {
	return cu.GetLocalAddr().NodeHash()
}

func (sh *CasperServerHandler) GetNode(ctx context.Context) (*core.IpfsNode, error) {
	repo, err := fsrepo.Open(sh.configRoot)
	if err != nil {
		return nil, err
	}

	return core.NewNode(ctx, &core.BuildCfg{Online: false, Repo: repo})
}

func (sh *CasperServerHandler) GetFileChecksum(ctx context.Context, uuid string, first, last int64, salt string) (string, error) {
	log.Debugf("Thrift: GetFileChecksum(%s, %d, %d, %s)", uuid, first, last, salt)

	//id := uid.UUIDToCid(base58.Decode(uuid))
	mhash, err := multihash.FromB58String(uuid)
	if err != nil {
		return "", err
	}
	id := cid.NewCidV0(mhash)

	n, err := sh.GetNode(ctx)
	if err != nil {
		return "", err
	}

	node, err := n.DAG.Get(ctx, id)
	if err != nil {
		return "", err
	}

	cs, err := val.ChecksumSalt(ctx, node, first, last, n.DAG, []byte(salt))
	if err != nil {
		return "", err
	}

	return cs.B58String(), nil
}

func (serverHandler *CasperServerHandler) Ping(ctx context.Context) (*casperproto.PingResult_, error) {
	log.Debugf("Thrift: Ping()")
	return &casperproto.PingResult_{
		Timestamp: time.Now().Unix(),
		ID:        serverHandler.NodeID(),
	}, nil
}

func (serverHandler *CasperServerHandler) SendUploadQuery(ctx context.Context, hash string, ipAddr string, size int64) (status string, err error) {
	log.Debugf("Thrift: SendUploadQuery(%s, %s, %d)", hash, ipAddr, size)

	var ipList []string
	err = json.Unmarshal([]byte(ipAddr), &ipList)
	if err != nil {
		return "", err
	}

	log.Debugf("Received peers: %v", ipList)

	///TODO: We might want to reimplement this without runCommand
	/// and return on first fail
	status, err = runCommand(ctx, []string{"files", "cp", "/ipfs/" + hash, "/"})
	status, err = runCommand(ctx, []string{"pin", "add", "--recursive", hash})
	//status, err = runCommand(ctx, []string{"cat", "/ipfs/" + hash})
	if err != nil {
		return "", err
	}

	c, _ := sc.GetContract()
	err = c.ConfirmUpload(serverHandler.NodeID(), hash, size)

	return
}

func (serverHandler *CasperServerHandler) SendDownloadQuery(ctx context.Context, hash string, ipAddr string, wallet string) (status string, err error) {
	log.Debugf("Thrift: SendDownloadQuery(%s, %s, %s)", hash, ipAddr, wallet)

	decision.AllowHash(hash, wallet)
	return "", nil
}

func (serverHandler *CasperServerHandler) SendDeleteQuery(ctx context.Context, hash string) (status string, err error) {
	log.Debugf("Thrift: SendDeleteQuery(%s)", hash)

	///TODO: We might want to reimplement this without runCommand
	status, err = runCommand(ctx, []string{"pin", "rm", "--recursive", hash})
	status, err = runCommand(ctx, []string{"files", "rm", "/" + hash})
	status, err = runCommand(ctx, []string{"files", "rm", "-r", "/" + hash})
	status, err = runCommand(ctx, []string{"block", "rm", hash})
	if err != nil {
		return "", err
	}

	c, _ := sc.GetContract()
	err = c.NotifySpaceFreed(serverHandler.NodeID(), hash, int64(commands.SizeOut))
	if err != nil {
		log.Error(err)
	}

	return "", err
}

func (serverHandler *CasperServerHandler) SendReplicationQuery(ctx context.Context, fileID string, nodeID string, size int64) (status string, err error) {
	log.Debugf("Thrift: SendReplicationQuery(%s, %s, %d)", nodeID, fileID, size)

	c, _ := sc.GetContract()
	verified, err := c.VerifyReplication(nodeID)
	if err != nil {
		return "", err
	}
	if verified {
		var peers []ma.Multiaddr
		peers, err = cu.GetPeersMultiaddrsByHash(fileID)
		if err != nil && len(peers) == 0 {
			return "", err
		}

		for _, peer := range peers {
			runCommand(ctx, []string{"swarm", "connect", peer.String()})
		}

		status, err = runCommand(ctx, []string{"files", "cp", "/ipfs/" + fileID, "/"})
		if err != nil {
			return
		}
		status, err = runCommand(ctx, []string{"pin", "add", "--recursive", fileID})
		if err != nil {
			return
		}

		///TODO: check actual size from network
		return "", c.ConfirmUpload(serverHandler.NodeID(), fileID, size)
	}
	return "", errors.New("replication verification failed")
}

func (serverHandler *CasperServerHandler) SendUpdateQuery(ctx context.Context, uuid string, hash string, size int64) (status string, err error) {
	log.Debugf("Thrift: SendUpdateQuery(%s, %s, %d)", uuid, hash, size)

	status, err = runCommand(ctx, []string{"upd", uuid, hash})
	if err != nil {
		return
	}

	h := uid.UUIDToHash(base58.Decode(uuid)).B58String()

	c, _ := sc.GetContract()
	err = c.ConfirmUpdate(serverHandler.NodeID(), h, size)
	return
}

// This is the func that every node invokes on CHECK INITIATOR other right after
// message with UUID has been appeared in SC logs
func (sh *CasperServerHandler) SendVerificationQuery(ctx context.Context, uuid string, ninfo *casperproto.NodeInfo) error {
	log.Debugf("Thrift: VerificationQuery(%s, %+v)", uuid, ninfo)

	addr, err := ipfsaddr.ParseString(ninfo.IpfsAddr)
	if err != nil {
		log.Error(err)
		return err
	}
	taddr, err := net.ResolveTCPAddr("tcp", ninfo.ThriftAddr)
	if err != nil {
		log.Error(err)
		return err
	}

	val.RegisterUUIDProvider(uuid, addr, taddr)

	return nil
}

func (sh *CasperServerHandler) SendChunkInfo(ctx context.Context, cinfo *casperproto.ChunkInfo) error {
	log.Debugf("Thrift: ChunkSectionInfo(%+v)", cinfo)
	go val.CollectResultsAndRespond(context.Background(), cinfo, sh.configRoot)
	log.Debugf("fihish ChunkSectionInfo()")
	return nil
}

func (sh *CasperServerHandler) SendChecksumHash(ctx context.Context, uuid string, ipfsAddr string, hashDiffuse string) error {
	log.Debugf("Thrift: SendChecksumHash(%s, %s)", uuid, ipfsAddr)
	addr, err := ipfsaddr.ParseString(ipfsAddr)
	if err != nil {
		return err
	}
	val.AddRound1Result(ctx, uuid, addr, hashDiffuse)
	log.Debugf("finish SendChecksumHash()")
	return nil
}

func (serverHandler *CasperServerHandler) SendValidationResults(ctx context.Context, uuid string, ipfsAddr string, addrToHash map[string]string) error {
	log.Debugf("Thrift: SendValidationResults(%s, %s)", uuid, ipfsAddr, addrToHash)

	// TODO Determine who is bad provider (if any) and send to SC

	return nil
}

func (serverHandler *CasperServerHandler) SendConnectQuery(ctx context.Context) (string, error) {
	log.Debugf("Thrift: SendConnectQuery")
	name, passwd := proxy.GenProxyCreds(26266637774 + time.Now().Unix()%179425859)
	return proxy.GetProxy(name, passwd)
}

func runCommand(ctx context.Context, args []string) (status string, err error) {
	var invoc cmdInvocation
	defer invoc.close()

	// parse the commandline into a command invocation
	parseErr := invoc.Parse(ctx, args)

	// ok now handle parse error (which means cli input was wrong,
	// e.g. incorrect number of args, or nonexistent subcommand)
	if parseErr != nil {
		// panic(parseErr.Error())
		printErr(parseErr)
		return "smells like ebola", parseErr
	}

	// ok, finally, run the command invocation.
	intrh, ctx := invoc.SetupInterruptHandler(ctx)
	defer intrh.Close()
	output, err := invoc.Run(ctx)
	if err != nil {
		printErr(err)
		return "smells like ebola", err
	}

	// everything went better than expected :)
	_, err = io.Copy(os.Stdout, output)
	if err != nil {
		printErr(err)
		return "smells like ebola", err
	}

	return "Dis is da wei", nil
}
