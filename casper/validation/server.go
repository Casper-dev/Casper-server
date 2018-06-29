package validation

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	thrift "github.com/Casper-dev/Casper-server/casper/thrift"
	uid "github.com/Casper-dev/Casper-server/casper/uuid"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/repo/fsrepo"
	"github.com/Casper-dev/Casper-thrift/casperproto"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	"gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"
	//"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
)

type RunningCheck struct {
	info    *casperproto.ChunkInfo
	round   *round
	nodes   []*cu.ExternalAddr
	results map[string]string
	mtx     sync.Mutex
}

func NewRunningCheck(cinfo *casperproto.ChunkInfo) *RunningCheck {
	return &RunningCheck{
		info:    cinfo,
		mtx:     sync.Mutex{},
		nodes:   make([]*cu.ExternalAddr, 0, NumChunkStoringNodes),
		results: make(map[string]string, NumChunkStoringNodes),
	}
}

// round is like sync.Cond but also performs Broadcast after specified timeout
type round struct {
	cond   *sync.Cond
	timer  *time.Timer
	result interface{}
}

func newRound(timeout time.Duration) *round {
	r := &round{cond: sync.NewCond(&sync.Mutex{})}
	r.timer = time.AfterFunc(timeout, r.Broadcast)
	return r
}

func (r *round) Wait() {
	r.cond.Wait()
}

func (r *round) Broadcast() {
	if r.timer != nil && !r.timer.Stop() {
		<-r.timer.C
	}
	r.cond.Broadcast()
}

var uuidProvMap = &sync.Map{}

const NumChunkStoringNodes = 2

//const sendChunkInfoTimeout = 1 * time.Minute
const sendChunkInfoTimeout = 10 * time.Second
const sendChecksumTimeout = 10 * time.Second
const sendVerificationQueryTimeout = 10 * time.Second
const round1Timeout = 5 * time.Minute

const validateBlockSize int64 = 1024
const diffuseLength = 16

func PerformValidation(ctx context.Context, n *core.IpfsNode, uuid string) error {
	rc := NewRunningCheck(nil)
	uuidProvMap.Store(uuid, rc)
	defer uuidProvMap.Delete(uuid)

	node, err := n.DAG.Get(ctx, uid.UUIDToCid(base58.Decode(uuid)))
	if err != nil {
		return err
	}

	rc.info, err = getRandomChunk(ctx, n, node, uuid, validateBlockSize)
	if err != nil {
		return err
	}

	log.Debugf("Random chunk %v %d %d", rc.info.UUID, rc.info.First, rc.info.Last)

	localAddr := cu.GetLocalAddr()
	rc.nodes = append(rc.nodes, localAddr)
	rc.info.Providers = append(rc.info.Providers, &casperproto.NodeInfo{IpfsAddr: localAddr.String(), ThriftAddr: localAddr.Thrift().String()})

	uuidProvMap.Store(uuid, rc)
	defer uuidProvMap.Delete(uuid)

	log.Info("Sleep for %s", sendVerificationQueryTimeout)
	time.Sleep(sendVerificationQueryTimeout)
	log.Debugf("Nodes after sleep: %+v", rc.nodes)

	// TODO: Call SC and initiate checking of specified uuid

	// Perform SendChunkInfo calls on all providers
	//rc.round = newRound(sendChunkInfoTimeout)
	wg := &sync.WaitGroup{}
	for _, prov := range rc.nodes[1:] {
		log.Debugf("SendChunkInfo to %s", prov)
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			thrift.RunClientClosure(addr, func(c *thrift.ThriftClient) (interface{}, error) {
				return nil, c.SendChunkInfo(ctx, rc.info)
			})
		}(prov.Thrift().String())
	}
	wg.Wait()
	//rc.round.Wait()

	// Send all nodes checksum
	salt := getSalt(rc.info.Diffuse, n.Identity.String())
	cs, err := ChecksumSalt(ctx, node, rc.info.First, rc.info.Last, n.DAG, salt)
	log.Debugf("Calculated checksum:", cs.B58String())
	//rc.round = newRound(sendChecksumTimeout)
	for _, prov := range rc.nodes {
		log.Debugf("SendChecksumHash to %s", prov)
		go func(addr string) {
			thrift.RunClientClosure(addr, func(c *thrift.ThriftClient) (interface{}, error) {
				return nil, c.SendChecksumHash(ctx, rc.info.UUID, localAddr.String(), cs.B58String())
			})
		}(prov.Thrift().String())
	}
	time.Sleep(sendChecksumTimeout)
	//rc.round.Wait()

	// Wait until results all results arrive
	//rc.round = newRound(round1Timeout)
	// Perform a call to SC

	return nil
}

func RegisterUUIDProvider(uuid string, ipfsAddr ipfsaddr.IPFSAddr, tAddr net.Addr) {
	log.Debugf("Dumping current UUID map")
	uuidProvMap.Range(func(k, v interface{}) bool {
		log.Debugf("%s: %s", k, v)
		return true
	})
	if v, ok := uuidProvMap.Load(uuid); ok {
		log.Debugf("Store UUID %s, addr %s %s", ipfsAddr, tAddr)
		rc := v.(*RunningCheck)
		rc.mtx.Lock()

		rc.nodes = append(rc.nodes, &cu.ExternalAddr{ipfsAddr, tAddr.(*net.TCPAddr)})
		ninfo := casperproto.NodeInfo{IpfsAddr: ipfsAddr.String(), ThriftAddr: tAddr.String()}
		rc.info.Providers = append(rc.info.Providers, &ninfo)
		if len(rc.info.Providers) == NumChunkStoringNodes { // all nodes except current
			log.Debugf("Received info about UUID %s for %d nodes", uuid, NumChunkStoringNodes)
			//rc.round.Broadcast()
		}
		fmt.Println(rc.info.Providers)

		rc.mtx.Unlock()
	}
}

func AddRound1Result(ctx context.Context, uuid string, ipfsAddr ipfsaddr.IPFSAddr, hashDiffuse string) {
	if v, ok := uuidProvMap.Load(uuid); ok {
		rc := v.(*RunningCheck)
		rc.mtx.Lock()

		log.Debugf("Old results: %+v", rc.results)
		rc.results[ipfsAddr.ID().Pretty()] = hashDiffuse
		log.Debugf("New results: %+v", rc.results)
		if len(rc.results) == NumChunkStoringNodes {
			//	rc.round.Broadcast()
		}

		rc.mtx.Unlock()
	} else {
		log.Debugf("No value stored for UUID %s", uuid)
	}
}

func CollectResultsAndRespond(ctx context.Context, cinfo *casperproto.ChunkInfo, configRoot string) {
	rc := NewRunningCheck(cinfo)
	uuidProvMap.Store(cinfo.UUID, rc)
	defer uuidProvMap.Delete(cinfo.UUID)

	repo, err := fsrepo.Open(configRoot)
	if err != nil {
		return
	}
	log.Debugf("Got repo")

	n, err := core.NewNode(ctx, &core.BuildCfg{Online: false, Repo: repo})
	if err != nil {
		return
	}
	log.Debugf("Got core node")

	id := uid.UUIDToCid(base58.Decode(cinfo.UUID))
	node, err := n.DAG.Get(ctx, id)
	if err != nil {
		return
	}
	log.Debugf("Got ipld node")

	salt := getSalt(cinfo.Diffuse, n.Identity.String())
	cs, err := ChecksumSalt(ctx, node, cinfo.First, cinfo.Last, n.DAG, salt)
	log.Debugf("Checksum:", cs.B58String())
	if err != nil {
		return
	}

	for _, prov := range cinfo.Providers {
		addr, err := ipfsaddr.ParseString(prov.IpfsAddr)
		if err != nil {
			log.Error(err)
			continue
		}
		taddr, err := net.ResolveTCPAddr("tcp", prov.ThriftAddr)
		if err != nil {
			log.Error(err)
			continue
		}
		rc.nodes = append(rc.nodes, &cu.ExternalAddr{addr, taddr})
	}

	rc.results[n.Identity.Pretty()] = cs.B58String()
	log.Debugf("Sleep chunk info: %s", sendChunkInfoTimeout)
	time.Sleep(sendChunkInfoTimeout)

	localAddr := cu.GetLocalAddr()
	//rc.round = newRound(sendChecksumTimeout)
	for _, prov := range rc.nodes {
		go func(addr string) {
			thrift.RunClientClosure(addr, func(c *thrift.ThriftClient) (interface{}, error) {
				return nil, c.SendChecksumHash(ctx, cinfo.UUID, localAddr.IPFS().String(), cs.B58String())
			})
		}(prov.Thrift().String())
	}
	//rc.round.Wait()
	time.Sleep(sendChecksumTimeout)

	// TODO send only to first
	thrift.RunClientClosure(rc.nodes[0].Thrift().String(), func(c *thrift.ThriftClient) (interface{}, error) {
		return nil, c.SendValidationResults(ctx, cinfo.UUID, localAddr.String(), rc.results)
	})
}

func getRandomChunk(ctx context.Context, n *core.IpfsNode, node node.Node, uuid string, blocksize int64) (*casperproto.ChunkInfo, error) {
	size, err := GetFilesize(node)
	if err != nil {
		return nil, err
	}

	rind := rand.Int63n((int64(size) / blocksize) + 1)
	first := rind * blocksize
	last := (rind + 1) * blocksize
	if last > int64(size) {
		last = int64(size)
	}

	diffuse := make([]byte, diffuseLength)
	rand.Read(diffuse)
	return &casperproto.ChunkInfo{
		UUID:      uuid,
		First:     first,
		Last:      last,
		Providers: make([]*casperproto.NodeInfo, 0, NumChunkStoringNodes),
		Diffuse:   string(diffuse),
	}, nil
}

func getSalt(a, b string) []byte {
	return []byte{}
	return []byte(a + b)
}
