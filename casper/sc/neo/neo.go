package neo

import (
	"context"
	"errors"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	sc "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"

	rpc "github.com/CityOfZion/neo-go/pkg/rpc"
	wallet "github.com/CityOfZion/neo-go/pkg/wallet"
)

const (
	roleNormal = 1
	roleBanned = 2
)

// ChainName is used in config
const ChainName = "NEO"

type nodeInfo struct {
	size   int64
	free   int64
	ipaddr string
	thrift string
	role   int
}

type Contract struct {
	contract string
	endpoint string
	neonAPI  string
	rpc      *rpc.Client
	wif      *wallet.WIF
}

const (
	defaultWIFKey   = "KxDgvEKzgSBPPfuVfw67oPQBSjidEiqTHURKSDL1R7yGaGYAeYnr"
	defaultContract = "c2fd3d71d1e0caa18092a65f5529a5cf87b8e799"
)

var _ sc.CasperSC = &Contract{}
var errNotImplemented = errors.New("not implemented")

func (c *Contract) Init(ctx context.Context, opts sc.InitOpts) (err error) {
	var gateway, wif, neonapi string
	var ok bool
	if gateway, ok = opts["Gateway"].(string); !ok {
		return errors.New("must provide Gateway")
	}
	if wif, ok = opts["WIF"].(string); !ok {
		return errors.New("must provide WIF")
	}
	if neonapi, ok = opts["NeonAPI"].(string); !ok {
		return errors.New("must provide NeonAPI")
	}

	c.wif, err = wallet.WIFDecode(wif, wallet.WIFVersion)
	if err != nil {
		return err
	}

	c.endpoint = gateway
	c.neonAPI = neonapi
	c.rpc, err = rpc.NewClient(ctx, c.endpoint, rpc.ClientOptions{})
	if err != nil {
		return err
	}
	if err := c.rpc.Ping(); err != nil {
		return fmt.Errorf("cant connect to NEO RPC server: %v", err)
	}

	if c.contract, ok = opts["ContractAddress"].(string); !ok {
		c.contract = defaultContract
	}

	return err
}

func (c *Contract) Initialized() bool {
	return c.rpc != nil && c.wif != nil
}

func (c *Contract) GetWallet() string {
	w, _ := c.wif.PrivateKey.Address()
	return w
}

func (c *Contract) AddToken(amount int64) error {
	res, err := c.callContractMethod("addtoken", amount)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) ConfirmDownload() error {
	// TODO refactor when this method will do anything useful
	_, err := c.callContractMethod("confirmdownload")
	return err
}

func (c *Contract) ConfirmUpdate(nodeID string, fileID string, size int64) error {
	res, err := c.callContractMethod("confirmupdate", nodeID, fileID, size)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) ConfirmUpload(nodeID string, fileID string, size int64) error {
	res, err := c.callContractMethod("confirmupload", nodeID, fileID, size)
	if err != nil {
		return err
	}

	return c.performTransaction(res)
}

func (c *Contract) GetFile(nodeID string, number int64) (name string, size int64, err error) {
	res, err := c.callContractMethod("getfile", nodeID, number)
	if err != nil {
		return "", 0, err
	}
	//if err = c.performTransaction(res); err != nil {
	//	return "", 0, err
	//}

	err = c.parseScriptResponse(res, &name, &size)
	return name, size, err
}

func (c *Contract) GetRPCAddr(nodeID string) (string, error) {
	n, err := c.getNodeInfo(nodeID)
	if err != nil {
		return "", err
	}
	return n.thrift, nil
}

func (c *Contract) getNodeInfo(nodeID string) (*nodeInfo, error) {
	res, err := c.callContractMethod("getinfo", nodeID)
	if err != nil {
		return nil, err
	}

	var size, free int64
	var rawip, rawth string
	var role = make([]byte, 1)
	err = c.parseScriptResponse(res, &size, &free, &rawip, &rawth, &role)
	if err != nil {
		return nil, err
	}

	n := &nodeInfo{
		size:   size,
		free:   free,
		ipaddr: string(rawip),
		thrift: string(rawth),
		role:   int(role[0]),
	}
	spew.Dump(n)
	return n, nil
}

func (c *Contract) GetNodeHash(nodeID string) (string, error) {
	return nodeID, nil
}

func (c *Contract) GetNumberOfFiles(nodeID string) (int64, error) {
	res, err := c.callContractMethod("getfilesnumber", nodeID)
	if err != nil {
		return 0, err
	}

	var n int64
	if err = c.parseScriptResponse(res, &n); err != nil {
		return 0, err
	}

	return n, nil
}

func (c *Contract) GetPeers(size int64, count int) ([]string, error) {
	res, err := c.callContractMethod("getpeers", size, count)
	if err != nil {
		return nil, err
	}

	ids := make([]string, count)
	ptrs := make([]interface{}, count)
	for i := range ids {
		ptrs[i] = &ids[i]
	}
	if err = c.parseScriptResponse(res, ptrs...); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c *Contract) GetPingTarget(nodeID string) (string, bool, error) {
	res, err := c.callContractMethod("getpingtarget", nodeID)
	if err != nil {
		return "", false, err
	}
	target := ""
	if err = c.parseScriptResponse(res, &target); err != nil {
		return "", false, err
	}
	return target, false, nil
}

func (c *Contract) GetAPIAddr(nodeID string) (string, error) {
	n, err := c.getNodeInfo(nodeID)
	if err != nil {
		return "", err
	}
	return n.ipaddr, nil
}

func (c *Contract) IsPrepaid(address string) (bool, error) {
	_, err := c.callContractMethod("isprepaid", address)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (c *Contract) NotifyDelete(nodeID string, fileID string, size int64) error {
	res, err := c.callContractMethod("notifydelete", nodeID, fileID, size)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) NotifySpaceFreed(nodeID string, fileID string, size int64) error {
	res, err := c.callContractMethod("notifyspacefreed", nodeID, fileID, size)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) NotifyVerificationTarget(nodeID string, fileID string) error {
	res, err := c.callContractMethod("notifyverificationtarget", nodeID, fileID)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) PrePay(amount int64) error {
	_, err := c.callContractMethod("prepay", amount)
	return err
}

func (c *Contract) RegisterProvider(nodeID string, telegram string, ipAddr string, thriftAddr string, size int64) error {
	res, err := c.callContractMethod("register", nodeID, size, ipAddr, thriftAddr, telegram)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) SendPingResult(nodeID string, success bool) (bool, error) {
	res, err := c.callContractMethod("sendpingresult", nodeID, success)
	if err != nil {
		return false, err
	}

	banned := false
	if err = c.parseScriptResponse(res, &banned); err != nil {
		return false, err
	}
	return banned, c.performTransaction(res)
}

func (c *Contract) ShowStoringPeers(fileID string) (ret []string, err error) {
	res, err := c.callContractMethod("showstoringpeers", fileID)
	if err != nil {
		return nil, err
	}

	ret = make([]string, 4)
	err = c.parseScriptResponse(res, &ret[0], &ret[1], &ret[2], &ret[3])
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Contract) SetAPIAddr(nodeID string, addr string) error {
	// TODO implement 'updateapiaddr' in SC
	return nil

	res, err := c.callContractMethod("updateapiaddr", nodeID, addr)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) SetRPCAddr(nodeID string, addr string) error {
	res, err := c.callContractMethod("updateipport", nodeID, addr)
	if err != nil {
		return err
	}
	return c.performTransaction(res)
}

func (c *Contract) VerifyReplication(nodeID string) (bool, error) {
	_, err := c.callContractMethod("verifyreplication", nodeID)
	return false, err
}

func (c *Contract) SetOriginCode(nodeID, originCode string) error {
	///TODO: implement on NEO
	return errNotImplemented
}

func (c *Contract) SubscribeVerificationTarget(ctx context.Context, callback sc.VerificationTargetFunc) error {
	return errNotImplemented
}

func (c *Contract) SubscribeConsensusResult(ctx context.Context, callback sc.ConsensusResultFunc) error {
	return errNotImplemented
}

func (c *Contract) SubscribeProviderCheck(ctx context.Context, callback sc.ProviderCheckFunc) error {
	return errNotImplemented
}
