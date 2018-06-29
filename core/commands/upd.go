package commands

import (
	"fmt"
	util "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	"io"
	"strings"

	"github.com/Casper-dev/Casper-server/blocks/blockstore"
	"github.com/Casper-dev/Casper-server/blockservice"
	cmds "github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-server/exchange/offline"
	dag "github.com/Casper-dev/Casper-server/merkledag"
	path "github.com/Casper-dev/Casper-server/path"
	pin "github.com/Casper-dev/Casper-server/pin"
)

var UpdCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Updates file corresponding to specified UUID.",
		ShortDescription: `
Recursively downloads blocks and pins them to local storage.
`,
		LongDescription: `
Makes DAG node with UUID <uuid> contain <ipfs-path>.
<ipfs-path> is recursively downloaded and pinned to local storage.
`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("uuid", true, false, "Base58 encoded UUID."),
		cmds.StringArg("ipfs-path", true, false, "The path to the IPFS object."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[1])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, _, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		pn := obj.(*dag.ProtoNode)

		addblockstore := blockstore.NewGCBlockstore(n.BaseBlocks, n.GCLocker)
		exch := offline.Exchange(addblockstore)
		bserv := blockservice.New(addblockstore, exch)
		dserv := dag.NewDAGService(bserv)

		defer n.Blockstore.PinLock().Unlock()

		// TODO: make an option to disable this behaviour
		oldCid := pn.Cid()
		log.Debugf("Remove pin on CID: %s", oldCid.String())
		n.Pinning.RemovePinWithMode(oldCid, pin.Recursive)

		pn.SetUUID(base58.Decode(req.Arguments()[0]))

		rnk, err := dserv.Add(pn)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		log.Debugf("Pin CID: %s", rnk.String())
		n.Pinning.PinWithMode(rnk, pin.Recursive)

		if err = n.Pinning.Flush(); err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(pn)
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			pn, ok := res.Output().(*dag.ProtoNode)
			if !ok {
				return nil, util.ErrCast()
			}

			return strings.NewReader(fmt.Sprintf("UUID: %s\nHash: %s\n",
				base58.Encode(pn.UUID()),
				pn.Cid().String(),
			)), nil
		},
	},
	Type: dag.ProtoNode{},
}
