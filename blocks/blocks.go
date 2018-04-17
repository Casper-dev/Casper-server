// Package blocks implements interface blocks.Block from go-block-format
// augmented with UUID.
// A block is raw data accompanied by a CID. The CID contains the multihash
// corresponding to the block.
package blocks

import (
	"fmt"
	"runtime/debug"

	"gitlab.com/casperDev/Casper-server/casper/uuid"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	blocks "gx/ipfs/QmSn9Td7xgxm9EV7iEjTckpUWmWApggzPxu7eFGWkkpwin/go-block-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

var log = logging.Logger("blocks")

// A BasicBlock is a singular block of data in ipfs. It implements the Block
// interface.
type BasicBlock struct {
	cid  *cid.Cid
	data []byte
	uuid []byte
}

// SplitData extracts UUID and Data from byte slice
func SplitData(rawdata []byte) (uid []byte, data []byte) {
	if len(rawdata) < uuid.UUIDLen {
		panic(fmt.Sprintf("Data is too small and has no UUID %x", rawdata))
	}

	return rawdata[:uuid.UUIDLen], rawdata[uuid.UUIDLen:]
}

// NewBlock creates a Block object from opaque data. It will hash the data.
func NewBlock(data []byte) *BasicBlock {
	cid := cid.NewCidV0(u.Hash(data))

	log.Debugf("NewBlock of length %d and with CID %s", len(data), cid.String())

	return &BasicBlock{data: data, cid: cid}
}

func NewBlockWithUUID(data []byte) *BasicBlock {
	var id *cid.Cid

	uid, d := SplitData(data)
	if uuid.IsUUIDNull(uid) {
		uid = nil
		id = cid.NewCidV0(u.Hash(data))
	} else {
		id = cid.NewCidV0(u.Hash(uid))
	}

	log.Debugf("NewBlockWithUUID of length %d with CID %s", len(d), id.String())

	// Do not store UUID if it is zero
	return &BasicBlock{data: d, uuid: uid, cid: id}
}

// NewBlockWithCid creates a new block when the hash of the data
// is already known, this is used to save time in situations where
// we are able to be confident that the data is correct.
func NewBlockWithCid(data []byte, c *cid.Cid) (*BasicBlock, error) {
	id, d := SplitData(data)
	log.Debugf("NewBlockWithCid: splitted uuid %d + %d %s", len(id), len(d), base58.Encode(id))

	if u.Debug {
		var rd = id
		if uuid.IsUUIDNull(id) {
			rd = d
		}
		chkc, err := c.Prefix().Sum(rd)

		if err != nil {
			return nil, err
		}

		if !chkc.Equals(c) {
			log.Debugf("Calculated hash %s != Argument hash %s", chkc.String(), c.String())
			debug.PrintStack()
			return nil, blocks.ErrWrongHash
		}
	}

	return &BasicBlock{data: d, uuid: id, cid: c}, nil
}

// Multihash returns the hash contained in the block CID.
func (b *BasicBlock) Multihash() mh.Multihash {
	return b.cid.Hash()
}

// RawData returns the block raw contents as a byte slice.
func (b *BasicBlock) RawData() []byte {
	if b.uuid != nil {
		return append(b.uuid, b.data...)
	}

	return append(uuid.NullUUID, b.data...)
}

// UUID returns the uuid of the block
func (b *BasicBlock) UUID() []byte {
	if b.uuid != nil {
		return b.uuid
	}

	return uuid.NullUUID
}

// Cid returns the content identifier of the block.
func (b *BasicBlock) Cid() *cid.Cid {
	return b.cid
}

// String provides a human-readable representation of the block CID.
func (b *BasicBlock) String() string {
	return fmt.Sprintf("[Block %s, UUID: %s]", b.Cid(), base58.Encode(b.UUID()))
}

// Loggable returns a go-log loggable item.
func (b *BasicBlock) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"block": b.Cid().String(),
		"data":  b.data,
		"uuid":  b.uuid,
	}
}
