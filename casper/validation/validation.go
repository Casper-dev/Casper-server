package validation

import (
	"context"
	"errors"

	bl "gitlab.com/casperDev/Casper-server/blocks"
	dag "gitlab.com/casperDev/Casper-server/merkledag"
	ft "gitlab.com/casperDev/Casper-server/unixfs"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

var log = logging.Logger("validator")

var ErrInvalidBoundaries = errors.New("Invalid boundaries")
var ErrNotFileNode = errors.New("Not a DAG file node")

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

const CSFuncCode = mh.SHA2_256

func CalcChecksum(data, salt []byte) mh.Multihash {
	cs, _ := mh.Sum(append(data, salt...), CSFuncCode, -1)
	return cs
}

func GetFilesize(n node.Node) (uint64, error) {
	switch v := n.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(v.Data())
		if err != nil {
			return 0, err
		}
		return fsn.FileSize(), nil
	case *dag.RawNode:
		_, data := bl.SplitData(v.RawData())
		return uint64(len(data)), nil
	default:
		return 0, ErrNotFileNode
	}
}

func GetSlice(ctx context.Context, n node.Node, start, stop uint64, serv node.NodeGetter) ([]byte, error) {
	if start > stop && stop != 0 {
		return nil, ErrInvalidBoundaries
	}
	switch v := n.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(v.Data())
		if err != nil {
			return nil, ErrNotFileNode
		}
		if fsn.Type == ft.TDirectory && len(fsn.Data) == 0 && len(v.Links()) == 1 {
			// this directory is wrapped over one file
			// return slice of that file
			child, err := v.Links()[0].GetNode(ctx, serv)
			if err != nil {
				return nil, err
			}
			return GetSlice(ctx, child, start, stop, serv)
		}
		if fsn.Type != ft.TFile && fsn.Type != ft.TRaw {
			return nil, ErrNotFileNode
		}
		if stop > fsn.FileSize() {
			return nil, ErrInvalidBoundaries
		} else if stop == 0 {
			stop = fsn.FileSize()
		}
		var buf []byte
		if l := uint64(len(fsn.Data)); l > 0 {
			log.Debugf("data node: len=%d, #links=%d", l, len(v.Links()))
			if stop < l {
				return fsn.Data[start:stop], nil
			}
			// TODO: check: data nodes have no links
			return fsn.Data[start:], nil
		}
		for i, bs := range fsn.BlockSizes() {
			log.Debugf("%02d: blocksize=%d, start=%d, stop=%d", i, bs, start, stop)
			if start >= bs {
				start -= bs
				stop -= bs
				continue
			}
			child, err := v.Links()[i].GetNode(ctx, serv)
			if err != nil {
				return nil, err
			}
			sl, err := GetSlice(ctx, child, start, min(stop, bs), serv)
			if err != nil {
				return nil, err
			}
			log.Debugf("got slice: len=%d:", len(sl))
			buf = append(buf, sl...)
			if stop <= bs {
				break
			}
			start = 0
			stop -= bs
		}
		return buf, nil
	case *dag.RawNode:
		_, data := bl.SplitData(v.RawData())
		l := uint64(len(data))
		if start < 0 || stop > l {
			return nil, ErrInvalidBoundaries
		}
		if stop == 0 {
			stop = l
		}
		return data[start:stop], nil
	}
	return nil, ErrNotFileNode
}

func Checksum(ctx context.Context, n node.Node, start, stop int64, serv node.NodeGetter) (mh.Multihash, error) {
	return ChecksumSalt(ctx, n, start, stop, serv, nil)
}
func ChecksumSalt(ctx context.Context, n node.Node, start, stop int64, serv node.NodeGetter, salt []byte) (mh.Multihash, error) {
	if start < 0 || stop < 0 {
		return nil, ErrInvalidBoundaries
	}

	data, err := GetSlice(ctx, n, uint64(start), uint64(stop), serv)
	if err != nil {
		return nil, err
	}
	if l := len(data); l != 0 {
		log.Debugf("Received data: %d fst=%x lst=%x", l, data[:1], data[l-1:])
	}
	return CalcChecksum(data, salt), nil
}
