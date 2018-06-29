package balanced

import (
	"errors"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"

	h "github.com/Casper-dev/Casper-server/importer/helpers"
)

var log = logging.Logger("balanced.builder")

func BalancedLayout(db *h.DagBuilderHelper, uuidList ...[]byte) (node.Node, error) {
	var offset uint64 = 0
	var root *h.UnixfsNode
	for level := 0; !db.Done(); level++ {

		nroot := db.NewUnixfsNode()
		db.SetPosInfo(nroot, 0)

		// add our old root as a child of the new root.
		if root != nil { // nil if it's the first node.
			if err := nroot.AddChild(root, db); err != nil {
				return nil, err
			}
		}

		// fill it up.
		if err := fillNodeRec(db, nroot, level, offset); err != nil {
			return nil, err
		}

		offset = nroot.FileSize()
		root = nroot
	}
	if root == nil {
		root = db.NewUnixfsNode()
	}

	if len(uuidList) > 0 {
		root.SetUUID(uuidList[0])
	}

	out, err := db.Add(root)
	if err != nil {
		return nil, err
	}

	err = db.Close()
	if err != nil {
		return nil, err
	}

	return out, nil
}

// fillNodeRec will fill the given node with data from the dagBuilders input
// source down to an indirection depth as specified by 'depth'
// it returns the total dataSize of the node, and a potential error
//
// warning: **children** pinned indirectly, but input node IS NOT pinned.
func fillNodeRec(db *h.DagBuilderHelper, node *h.UnixfsNode, depth int, offset uint64) error {
	if depth < 0 {
		return errors.New("attempt to fillNode at depth < 0")
	}

	// Base case
	if depth <= 0 { // catch accidental -1's in case error above is removed.
		child, err := db.GetNextDataNode()
		if err != nil {
			return err
		}

		node.Set(child)
		return nil
	}

	// while we have room AND we're not done
	for node.NumChildren() < db.Maxlinks() && !db.Done() {
		child := db.NewUnixfsNode()
		db.SetPosInfo(child, offset)

		err := fillNodeRec(db, child, depth-1, offset)
		if err != nil {
			return err
		}

		if err := node.AddChild(child, db); err != nil {
			return err
		}
		offset += child.FileSize()
	}

	return nil
}
