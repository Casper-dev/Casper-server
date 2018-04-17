package commands

import (
	"context"
	"fmt"
	"io"
	"net"

	cu "gitlab.com/casperDev/Casper-server/casper/casper_utils"
	"gitlab.com/casperDev/Casper-server/casper/crypto"
	"gitlab.com/casperDev/Casper-server/client"
	cmds "gitlab.com/casperDev/Casper-server/commands"
	core "gitlab.com/casperDev/Casper-server/core"
	coreunix "gitlab.com/casperDev/Casper-server/core/coreunix"

	"gitlab.com/casperDev/Casper-SC/casper_sc"

	"gx/ipfs/QmeWjRodbcZFKe5tMN7poEx3izym6osrLSnTLf9UjJZBbs/pb"
)

const progressBarMinSize = 1024 * 1024 * 8 // show progress bar for outputs > 8MiB

// defined in commands/add
//const passwordOptionName = "password"

var CatCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Show IPFS object data.",
		ShortDescription: "Displays the data contained by an IPFS or IPNS object(s) at the given path.",
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, false, "The path to the IPFS object(s) to be outputted.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.StringOption(passwordOptionName, "Password decryption key"),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		node, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		caller, _, _ := req.Option(cmds.CallerOpt).String()
		if caller == cmds.CallerOptClient {
			_, _, auth, _ := Casper_SC.GetSC()
			wallet := auth.From.String()
			hash := req.Arguments()[0]
			peers, err := cu.GetPeersMultiaddrsByHash(hash)
			if err != nil && len(peers) == 0 {
				res.SetError(err, cmds.ErrClient)
				return
			}
			for _, peer := range peers {
				err := node.ConnectToPeer(req.Context(), peer.String())
				if err != nil {
					log.Error("Failed to connect: %s", err)
					continue
				}
				addr, _ := cu.MultiaddrToTCPAddr(peer)
				thriftAddr := net.JoinHostPort(addr.String(), "9090")
				err = client.HandleClientDownload(req.Context(), thriftAddr, hash, wallet)
				if err == nil {
					break
				}
			}
			fmt.Println("Success!")
		}

		if !node.OnlineMode() {
			if err := node.SetupOfflineRouting(); err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}

		readers, length, err := cat(req.Context(), node, req.Arguments())
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		/*
			if err := corerepo.ConditionalGC(req.Context(), node, length); err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		*/

		res.SetLength(length)
		reader := io.MultiReader(readers...)

		res.SetOutput(reader)
	},
	PostRun: func(req cmds.Request, res cmds.Response) {
		if res.Error() != nil {
			return
		}
		reader := res.Output().(io.Reader)
		if password, pwdset, _ := req.Option(passwordOptionName).String(); pwdset {
			reader = crypto.NewAESReader(reader, []byte(password))
		}

		var bar *pb.ProgressBar
		if res.Length() >= progressBarMinSize {
			bar, reader = progressBarForReader(res.Stderr(), reader, int64(res.Length()))
			bar.Start()
		}

		res.SetOutput(reader)
	},
}

func cat(ctx context.Context, node *core.IpfsNode, paths []string) ([]io.Reader, uint64, error) {
	readers := make([]io.Reader, 0, len(paths))
	length := uint64(0)
	for _, fpath := range paths {
		read, err := coreunix.Cat(ctx, node, fpath)
		if err != nil {
			return nil, 0, err
		}
		readers = append(readers, read)
		length += uint64(read.Size())
	}
	return readers, length, nil
}
