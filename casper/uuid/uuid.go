package uuid

import (
	"bytes"

	uuid "github.com/satori/go.uuid"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

const UUIDLen = 16

var NullUUID = make([]byte, UUIDLen)

func IsUUIDNull(u []byte) bool {
	return u == nil || bytes.Equal(u, NullUUID)
}

func GenUUID() []byte {
	u, _ := uuid.NewV4()
	return u.Bytes()
}

func UUIDToHash(uuid []byte) multihash.Multihash {
	hash, _ := multihash.Sum(uuid, multihash.SHA2_256, -1)
	return hash
}

func UUIDToCid(uuid []byte) *cid.Cid {
	mhash, _ := multihash.Sum(uuid, multihash.SHA2_256, -1)
	return cid.NewCidV0(mhash)
}
