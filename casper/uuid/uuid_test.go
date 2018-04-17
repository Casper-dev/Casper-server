package uuid_test

import (
	"testing"

	"gitlab.com/casperDev/Casper-server/casper/uuid"

	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
)

func TestUUIDToHash(t *testing.T) {
	testCases := []struct{ uuid, hash string }{
		{"CTxVEi21hGKWz1ptGH7DUJ", "QmW6xj9bDgNBapeaW4WVuXEvLpriGLGbdVHtRmwQwwdVkK"},
		{"SyBiYPJo2yMFG9bJr45DBs", "QmXWZSRFViFme2o7oagVRKr97jBaxNrFvQ6NUffC8ytu4P"},
	}

	for _, c := range testCases {
		hash, err := uuid.UUIDToHash(base58.Decode(c.uuid))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if hash != c.hash {
			t.Fatalf("Expected: %s, got %s", c.hash, hash)
		}
	}
}
