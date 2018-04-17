package validation_test

import (
	"bytes"
	"context"
	"testing"

	"fmt"

	bstest "gitlab.com/casperDev/Casper-server/blockservice/test"
	"gitlab.com/casperDev/Casper-server/casper/validation"
	imp "gitlab.com/casperDev/Casper-server/importer"
	chunk "gitlab.com/casperDev/Casper-server/importer/chunk"
	dag "gitlab.com/casperDev/Casper-server/merkledag"
)

func slicesEqual(a, b []byte) (bool, string) {
	l := len(a)
	if l != len(b) {
		return false, fmt.Sprintf("Slice's lenths differ: %d != %d", len(a), len(b))
	}
	for i, c := range a {
		if c != b[i] {
			return false, fmt.Sprintf("Slices differ at byte %d: %02d != %02d", i, c, b[i])
		}
	}

	return true, ""
}

func TestGetSlice(t *testing.T) {
	bsi := bstest.Mocks(1)
	ds := dag.NewDAGService(bsi[0])

	bsize := uint64(512)
	l := 2048 * int(bsize) // 1 MiB

	var buf bytes.Buffer
	buf.Grow(l)
	for i := 0; i < l; i++ {
		buf.WriteByte(byte(i % 0x100))
	}
	fullb := buf.Bytes()

	// if there will be unexpected errors, try use bytes.NewReader(fullb)
	file, err := imp.BuildDagFromReader(ds, chunk.NewSizeSplitter(&buf, int64(bsize)))
	if err != nil {
		t.Fatal(err)
	}

	// Get full file
	s, _ := validation.GetSlice(context.Background(), file, 0, 0, ds)
	if eq, r := slicesEqual(s, fullb); !eq {
		t.Fatalf(r)
	}

	// Get slice containing multiple blocks from the middle
	s, err = validation.GetSlice(context.Background(), file, bsize*10, bsize*12, ds)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if eq, r := slicesEqual(s, fullb[:bsize*2]); !eq {
		t.Fatalf(r)
	}

	// Get slice containing block boundary
	s, _ = validation.GetSlice(context.Background(), file, bsize*3-2, bsize*3+2, ds)
	if eq, r := slicesEqual(s, []byte{0xFE, 0xFF, 0x00, 0x01}); !eq {
		t.Fatalf(r)
	}

	// Provide wrong boundaries
	_, err = validation.GetSlice(context.Background(), file, 2, 1, ds)
	if err != validation.ErrInvalidBoundaries {
		t.Fatalf("Unexpected error: %s", err)
	}

	// provide file length explicitly
	s, err = validation.GetSlice(context.Background(), file, 0, uint64(l), ds)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if eq, r := slicesEqual(s, fullb); !eq {
		t.Fatalf(r)
	}

	// provide extra length
	_, err = validation.GetSlice(context.Background(), file, 0, uint64(l+1), ds)
	if err != validation.ErrInvalidBoundaries {
		t.Fatalf("Unexpected error: %s", err)
	}
}
