package dagutils

import (
	"context"
	"fmt"

	mdag "gitlab.com/casperDev/Casper-server/merkledag"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
)

// DiffEnumerate fetches every object in the graph pointed to by 'to' that is
// not in 'from'. This can be used to more efficiently fetch a graph if you can
// guarantee you already have the entirety of 'from'
func DiffEnumerate(ctx context.Context, dserv node.NodeGetter, from, to *cid.Cid) error {
	fnd, err := dserv.Get(ctx, from)
	if err != nil {
		return fmt.Errorf("get %s: %s", from, err)
	}

	tnd, err := dserv.Get(ctx, to)
	if err != nil {
		return fmt.Errorf("get %s: %s", to, err)
	}

	diff := getLinkDiff(fnd, tnd)

	sset := cid.NewSet()
	for _, c := range diff {
		// Since we're already assuming we have everything in the 'from' graph,
		// add all those cids to our 'already seen' set to avoid potentially
		// enumerating them later
		if c.bef != nil {
			sset.Add(c.bef)
		}
	}
	for _, c := range diff {
		if c.bef == nil {
			if sset.Has(c.aft) {
				continue
			}
			err := mdag.EnumerateChildrenAsync(ctx, mdag.GetLinksDirect(dserv), c.aft, sset.Visit)
			if err != nil {
				return err
			}
		} else {
			err := DiffEnumerate(ctx, dserv, c.bef, c.aft)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// if both bef and aft are not nil, then that signifies bef was replaces with aft.
// if bef is nil and aft is not, that means aft was newly added
// if aft is nil and bef is not, that means bef was deleted
type diffpair struct {
	bef, aft *cid.Cid
}

// getLinkDiff returns a changeset between nodes 'a' and 'b'. Currently does
// not log deletions as our usecase doesnt call for this.
func getLinkDiff(a, b node.Node) []diffpair {
	ina := make(map[string]*node.Link)
	inb := make(map[string]*node.Link)
	var aonly []*cid.Cid
	for _, l := range b.Links() {
		inb[l.Cid.KeyString()] = l
	}
	for _, l := range a.Links() {
		var key = l.Cid.KeyString()
		ina[key] = l
		if inb[key] == nil {
			aonly = append(aonly, l.Cid)
		}
	}

	var out []diffpair
	var aindex int

	for _, l := range b.Links() {
		if ina[l.Cid.KeyString()] != nil {
			continue
		}

		if aindex < len(aonly) {
			out = append(out, diffpair{bef: aonly[aindex], aft: l.Cid})
			aindex++
		} else {
			out = append(out, diffpair{aft: l.Cid})
			continue
		}
	}
	return out
}
