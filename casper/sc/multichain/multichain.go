package multichain

import (
	"context"
	"errors"
	"fmt"

	neosc "github.com/Casper-dev/Casper-server/casper/sc/neo"
	sc "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"
	solsc "github.com/Casper-dev/Casper-server/casper/sc/solidity"
)

// ChainName is used in config
const ChainName = "Multi"

type Contract struct {
	neo sc.CasperSC
	eth sc.CasperSC
}

var _ sc.CasperSC = &Contract{}
var errNotImplemented = errors.New("not implemented")

func (c *Contract) Init(ctx context.Context, opts sc.InitOpts) (err error) {
	fmt.Printf("opts in Multi.Init(): %+v\n", opts)

	if c.neo == nil {
		c.neo = &neosc.Contract{}
	}
	if c.eth == nil {
		c.eth = &solsc.Contract{}
	}

	if err = c.neo.Init(ctx, opts["NEO"].(sc.InitOpts)); err != nil {
		return err
	}
	return c.eth.Init(ctx, opts["ETH"].(sc.InitOpts))
}

func (c *Contract) Initialized() bool {
	return c.neo != nil && c.neo.Initialized() &&
		c.eth != nil && c.eth.Initialized()
}

func (c *Contract) GetWallet() string {
	return c.eth.GetWallet()
}

func (c *Contract) AddToken(amount int64) error {
	c.neo.AddToken(amount)
	return c.eth.AddToken(amount)
}

func (c *Contract) ConfirmDownload() error {
	c.neo.ConfirmDownload()
	return c.eth.ConfirmDownload()
}

func (c *Contract) ConfirmUpdate(nodeID string, fileID string, size int64) error {
	c.neo.ConfirmUpdate(nodeID, fileID, size)
	return c.eth.ConfirmUpdate(nodeID, fileID, size)
}

func (c *Contract) ConfirmUpload(nodeID string, fileID string, size int64) error {
	c.neo.ConfirmUpload(nodeID, fileID, size)
	return c.eth.ConfirmUpload(nodeID, fileID, size)
}

func (c *Contract) GetFile(nodeID string, number int64) (name string, size int64, err error) {
	return c.eth.GetFile(nodeID, number)
}

func (c *Contract) GetRPCAddr(nodeID string) (string, error) {
	return c.eth.GetRPCAddr(nodeID)
}

func (c *Contract) GetNodeHash(nodeID string) (string, error) {
	return nodeID, nil
}

func (c *Contract) GetNumberOfFiles(nodeID string) (int64, error) {
	return c.eth.GetNumberOfFiles(nodeID)
}

func (c *Contract) GetPeers(size int64, count int) ([]string, error) {
	return c.eth.GetPeers(size, count)
}

func (c *Contract) GetPingTarget(nodeID string) (string, bool, error) {
	return c.eth.GetPingTarget(nodeID)
}

func (c *Contract) GetAPIAddr(nodeID string) (string, error) {
	return c.eth.GetAPIAddr(nodeID)
}

func (c *Contract) IsPrepaid(address string) (bool, error) {
	return c.eth.IsPrepaid(address)
}

func (c *Contract) NotifyDelete(nodeID string, fileID string, size int64) error {
	c.neo.NotifyDelete(nodeID, fileID, size)
	return c.eth.NotifyDelete(nodeID, fileID, size)
}

func (c *Contract) NotifySpaceFreed(nodeID string, fileID string, size int64) error {
	c.neo.NotifySpaceFreed(nodeID, fileID, size)
	return c.eth.NotifySpaceFreed(nodeID, fileID, size)
}

func (c *Contract) NotifyVerificationTarget(nodeID string, fileID string) error {
	c.neo.NotifyVerificationTarget(nodeID, fileID)
	return c.eth.NotifyVerificationTarget(nodeID, fileID)
}

func (c *Contract) PrePay(amount int64) error {
	c.neo.PrePay(amount)
	return c.eth.PrePay(amount)
}

func (c *Contract) RegisterProvider(nodeID string, telegram string, ipAddr string, thriftAddr string, size int64) error {
	c.neo.RegisterProvider(nodeID, telegram, ipAddr, thriftAddr, size)
	return c.eth.RegisterProvider(nodeID, telegram, ipAddr, thriftAddr, size)
}

func (c *Contract) SendPingResult(nodeID string, success bool) (bool, error) {
	c.neo.SendPingResult(nodeID, success)
	return c.eth.SendPingResult(nodeID, success)
}

func (c *Contract) ShowStoringPeers(fileID string) ([]string, error) {
	return c.eth.ShowStoringPeers(fileID)
}

func (c *Contract) SetAPIAddr(nodeID string, addr string) error {
	c.neo.SetAPIAddr(nodeID, addr)
	return c.eth.SetAPIAddr(nodeID, addr)
}

func (c *Contract) SetRPCAddr(nodeID string, addr string) error {
	c.neo.SetRPCAddr(nodeID, addr)
	return c.eth.SetRPCAddr(nodeID, addr)
}

func (c *Contract) VerifyReplication(nodeID string) (bool, error) {
	c.neo.VerifyReplication(nodeID)
	return c.eth.VerifyReplication(nodeID)
}

func (c *Contract) SetOriginCode(nodeID, originCode string) error {
	c.neo.SetOriginCode(nodeID, originCode)
	return c.eth.SetOriginCode(nodeID, originCode)
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
