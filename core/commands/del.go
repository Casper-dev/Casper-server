package commands

import (
	"fmt"
	"net"

	util "github.com/Casper-dev/Casper-server/blocks/blockstore/util"
	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	sc "github.com/Casper-dev/Casper-server/casper/sc"
	"github.com/Casper-dev/Casper-server/client"
	cmds "github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-server/core/corerepo"
)

var DelCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Remove block from server.",
		ShortDescription: "TODO",
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, false, "The path to the IPFS object to be removed.").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		caller, _, _ := req.Option(cmds.CallerOpt).String()
		if caller == cmds.CallerOptWeb {
			removed, err := corerepo.Unpin(n, req.Context(), req.Arguments(), true)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			ch, err := util.RmBlocks(n.Blockstore, n.Pinning, removed, util.RmBlocksOpts{
				Quiet: true,
				Force: true,
			})
			log.Debugf("Removed %d blocks", len(removed))
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			// wait for delete to finish
			for _ = range ch {
			}
			return
		}

		hash := req.Arguments()[0]
		c, _ := sc.GetContract()
		c.NotifyDelete(cu.GetLocalAddr().NodeHash(), hash, 1337)

		peers, err := cu.GetPeersMultiaddrsByHash(hash)
		if err != nil && len(peers) == 0 {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		for _, peer := range peers {
			err := n.ConnectToPeer(req.Context(), peer.String())
			if err != nil {
				log.Error("Failed to connect: %s", err)
				continue
			}
			addr, _ := cu.MultiaddrToTCPAddr(peer)
			thriftAddr := net.JoinHostPort(addr.IP.String(), "9090")
			err = client.HandleClientDelete(req.Context(), thriftAddr, hash)
			if err != nil {
				log.Errorf("Error while deleting file from peer '%s'", peer)
			}
		}

		fmt.Println("Success!")
	},
}
