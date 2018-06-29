package sc_interface

import "context"

// Events which can be handled asynchronously
// TODO probably will be removed
type VerificationTargetFunc = func(fileID string, id string)
type ConsensusResultFunc = func(fileID string, consensus [4][32]byte)
type ProviderCheckFunc = func()

type InitOpts = map[string]interface{}

// CasperSC interface must be implemented by every
// binding to blockchain.
// For example, see solidity binding in sc_solidity.go
// TODO think about wallet for auth
type CasperSC interface {
	// Init creates connection to RPC nodes (if any)
	// and does all necessary preparation, so that other
	// are able to perform interaction with SC
	Init(context.Context, InitOpts) error

	// Initialized checks if contract can perform interaction with SC
	Initialized() bool

	// GetWallet returns blockchain-specific wallet format
	// It is used as argument to other functions.
	GetWallet() string

	// AddToken add specified amount of tokens to the owner's wallet
	// TODO remove when deploy
	AddToken(amount int64) error

	// ConfirmDownload can be invoked by client after he/she has
	// successfully downloaded the file?
	// TODO what is the purpose of this function?
	ConfirmDownload() error

	// ConfirmUpdate is invoked by provider after it has successfully
	// updated file with specified id
	ConfirmUpdate(nodeID string, fileID string, size int64) error

	// ConfirmUpload is invoked by provider after it has successfully
	// uploaded file with specified id
	ConfirmUpload(nodeID string, fileID string, size int64) error

	// GetAPIAddr
	GetAPIAddr(nodeID string) (string, error)

	// GetFile
	GetFile(nodeID string, number int64) (string, int64, error)

	// GetNumberOfFiles
	GetNumberOfFiles(nodeID string) (int64, error)

	// GetPeers returns `count` random peers for storing file of size `size`
	GetPeers(size int64, count int) ([]string, error)

	// GetPingTarget is called by provider to receive target
	// for checking its availability
	GetPingTarget(nodeID string) (string, bool, error)

	// GetRPCAddr
	GetRPCAddr(nodeID string) (string, error)

	// IsPrepaid checks if client has prepaid file download
	IsPrepaid(address string) (bool, error)

	// NotifyDelete is called by provider after successfull deletion of file
	// TODO discuss parameters and use cases
	NotifyDelete(nodeID string, fileID string, size int64) error

	// NotifySpaceFreed
	// TODO what is the purpose of this function (why do we need both 2 and 3 arguments)?
	NotifySpaceFreed(nodeID string, fileID string, size int64) error

	// NotifyVerificationTarget is called by provider after start of verification
	NotifyVerificationTarget(nodeID string, fileID string) error

	// PrePay is invoked by client
	PrePay(amount int64) error

	// RegisterProvider is invoked by every new provider which wants to become
	// a part of Csper API
	RegisterProvider(nodeID string, telegram string, ipAddr string, thriftAddr string, size int64) error

	// SetOriginCode is invoked after RegisterProvider to bind node to it's geo location
	SetOriginCode(nodeID, originCode string) error

	// SendPingResult
	SendPingResult(nodeID string, success bool) (bool, error)

	// ShowStoringPeers return list of string's, which store
	// file with specified id
	ShowStoringPeers(fileID string) ([]string, error)

	// SetAPIAddr sets API address in SC
	SetAPIAddr(nodeID string, addr string) error

	// SetRPCAddr sets RPC address in SC
	SetRPCAddr(nodeID string, addr string) error

	// VerifyReplication returns true if node with given id
	// was banned or removed
	// TODO rename this method
	VerifyReplication(nodeID string) (bool, error)

	SubscribeVerificationTarget(ctx context.Context, callback VerificationTargetFunc) error
	SubscribeConsensusResult(ctx context.Context, callback ConsensusResultFunc) error
	SubscribeProviderCheck(ctx context.Context, callback ProviderCheckFunc) error
}
