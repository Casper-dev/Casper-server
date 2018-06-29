package solidity

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	sc "github.com/Casper-dev/Casper-server/casper/sc/sc_interface"

	"github.com/Casper-dev/Casper-SC/casper"
	"github.com/Casper-dev/Casper-SC/casper_sc"

	b58 "gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ChainName is used in config
const ChainName = "ETH"

// Type assertions
var _ sc.CasperSC = &Contract{}

type Contract struct {
	casper *casper.Casper
	eth    *ethclient.Client
	auth   *bind.TransactOpts
}

func (c *Contract) Init(ctx context.Context, opts sc.InitOpts) (err error) {
	var gateway, privkey string
	var ok bool
	if gateway, ok = opts["Gateway"].(string); !ok {
		return errors.New("must provide gateway")
	}
	if privkey, ok = opts["PrivateKey"].(string); !ok {
		return errors.New("must provide privatekey")
	}

	iopts := &Casper_SC.InitOpts{Gateway: gateway, PrivateKey: privkey}
	if addr, ok := opts["ContractAddress"].(string); ok {
		iopts.ContractAddress = addr
	}
	spew.Dump(opts)
	c.casper, c.eth, c.auth, err = Casper_SC.InitSC(ctx, iopts)
	return err
}

func (c *Contract) Initialized() bool {
	return Casper_SC.Initialized()
}

func (c *Contract) GetWallet() string {
	return c.auth.From.String()
}

func (c *Contract) AddToken(amount int64) error {
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.AddToken(c.auth, big.NewInt(amount))
	}, c.eth)

	return err
}

func (c *Contract) ConfirmDownload() error {
	_, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.ConfirmDownload(c.auth)
	}, c.eth)

	return err
}

func (c *Contract) ConfirmUpdate(nodeID string, fileID string, size int64) error {
	_, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.ConfirmUpdate(c.auth, nodeID, fileID, big.NewInt(size))
	}, c.eth)

	return err
}

func (c *Contract) ConfirmUpload(nodeID string, fileID string, size int64) error {
	_, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.ConfirmUpload(c.auth, nodeID, fileID, big.NewInt(size))
	}, c.eth)

	return err
}

func (c *Contract) GetFile(nodeID string, number int64) (string, int64, error) {
	fileID, size, err := c.casper.GetFile(nil, nodeID, big.NewInt(number))
	if err != nil {
		return "", 0, err
	}
	return fileID, size.Int64(), err
}

func (c *Contract) GetRPCAddr(nodeID string) (string, error) {
	addr, _, err := c.casper.GetNodeAddr(nil, nodeID)
	return addr, err
}

func (c *Contract) GetNumberOfFiles(nodeID string) (int64, error) {
	n, err := c.casper.GetNumberOfFiles(nil, nodeID)
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func (c *Contract) GetPeers(size int64, count int) (peers []string, err error) {
	for i := 0; i < count; i += 4 {
		p1, p2, p3, p4, err := c.casper.GetPeers(nil, big.NewInt(size), big.NewInt(time.Now().UnixNano()))
		if err != nil {
			return nil, err
		}
		peers = append(peers, p1, p2, p3, p4)
	}
	return peers, nil
}

func (c *Contract) GetPingTarget(nodeID string) (string, bool, error) {
	target, err := c.casper.GetPingTarget(nil)
	return target, false, err
}

func (c *Contract) GetStoringPeers(fileID string) (int, error) {
	n, err := c.casper.GetStoringPeers(nil, fileID)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func (c *Contract) GetAPIAddr(nodeID string) (string, error) {
	_, addr, err := c.casper.GetNodeAddr(nil, nodeID)
	return addr, err
}

func (c *Contract) IsPrepaid(address string) (bool, error) {
	return c.casper.IsPrepaid(nil, common.HexToAddress(address))
}

func (c *Contract) NotifyDelete(string, string, int64) error {
	//panic("not implemented")
	return nil
}

func (c *Contract) NotifySpaceFreed(nodeID string, fileID string, size int64) error {
	_, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.NotifySpaceFreed(c.auth, nodeID, fileID, big.NewInt(size))
	}, c.eth)

	return err
}

func (c *Contract) NotifyVerificationTarget(nodeID string, fileID string) error {
	_, err := c.casper.NotifyVerificationTarget(c.auth, fileID, nodeID)
	return err
}

func (c *Contract) PrePay(amount int64) error {
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.PrePay(c.auth, big.NewInt(amount))
	}, c.eth)

	return err
}

func (c *Contract) RegisterProvider(nodeID string, telegram string, ipAddr string, thriftAddr string, size int64) error {
	var telegramBytes [32]byte
	copy(telegramBytes[:], telegram)
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.RegisterProvider(c.auth, nodeID, telegramBytes, ipAddr, thriftAddr, big.NewInt(size))
	}, c.eth)

	return err
}

func (c *Contract) SetOriginCode(nodeID, originCode string) error {
	var originCode2 [4]byte
	copy(originCode2[:], []byte(originCode))
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.SetCountryCode(c.auth, nodeID, originCode2)
	}, c.eth)

	return err
}

func (c *Contract) RemoveProviderMachine(nodeID string) error {
	_, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.RemoveProviderMachine(c.auth, nodeID)
	}, c.eth)

	return err
}

func (c *Contract) SendPingResult(nodeID string, success bool) (bool, error) {
	result, err := Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return c.casper.SendPingResult(c.auth, nodeID, success)
	}, c.eth)

	return strings.Contains(result, "Banned!"), err
}

func (c *Contract) ShowStoringPeers(fileID string) ([]string, error) {
	p1, p2, p3, p4, err := c.casper.ShowStoringPeers(nil, fileID)
	if err != nil {
		return nil, err
	}
	return []string{p1, p2, p3, p4}, nil
}

func (c *Contract) SetAPIAddr(nodeID string, addr string) error {
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.SetAPIAddr(c.auth, nodeID, addr)
	}, c.eth)

	return err
}

func (c *Contract) SetRPCAddr(nodeID string, addr string) error {
	_, err := Casper_SC.ValidateMineTX(func() (*types.Transaction, error) {
		return c.casper.SetRPCAddr(c.auth, nodeID, addr)
	}, c.eth)

	return err
}

func (c *Contract) VerifyReplication(nodeID string) (bool, error) {
	return c.casper.IsDead(nil, nodeID)
}

func (c *Contract) SubscribeVerificationTarget(ctx context.Context, callback sc.VerificationTargetFunc) error {
	return Casper_SC.SubscribeToReplicationLogs(ctx, func(log *casper.CasperVerificationTarget) {
		callback(log.UUID, log.Id)
	})
}

func (c *Contract) SubscribeConsensusResult(ctx context.Context, callback sc.ConsensusResultFunc) error {
	return Casper_SC.SubscribeToConsensusLogs(ctx, func(log *casper.CasperConsensusResult) {
		callback(log.UUID, log.Consensus)
	})
}

func (c *Contract) SubscribeProviderCheck(ctx context.Context, callback sc.ProviderCheckFunc) error {
	return Casper_SC.SubscribeToProvidersCheckLogs(ctx, func(log *casper.CasperProviderCheckEvent) {
		callback()
	})
}

func hashToBytes(nodeID string) (bs [32]byte, err error) {
	dh, err := mh.Decode(b58.Decode(nodeID))
	if err != nil {
		return bs, err
	} else if dh.Code != mh.SHA2_256 {
		return bs, errors.New("invalid multihash (expected SHA2_256)")
	}

	// SHA2_256 digest has exactly 32 bytes in it
	copy(bs[:], dh.Digest[:])
	return bs, nil
}

func bytesToHash(id [32]byte) (string, error) {
	raw, _ := mh.Encode(id[:], mh.SHA2_256)
	mh, err := mh.Cast(raw)
	if err != nil {
		return "", err
	}
	return mh.B58String(), nil
}
