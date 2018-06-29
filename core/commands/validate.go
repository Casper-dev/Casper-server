package commands

import (
	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	thrift "github.com/Casper-dev/Casper-server/casper/thrift"
	val "github.com/Casper-dev/Casper-server/casper/validation"
	cmds "github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-thrift/casperproto"

	"gx/ipfs/QmX3U3YXCQ6UYBxq2LVWF8dARS1hPUTEYLrSx654Qyxyw6/go-multiaddr-net"
	"gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"
	"net"
)

var ValidateCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Validate specific UUID.",
		ShortDescription: `
Lists running and recently run commands.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("uuid", true, false, "UUID to validate"),
	},
	Options: []cmds.Option{
		cmds.StringOption("server", "s", "Perform validation as client"),
		cmds.StringOption("node", "Node ID"),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		id := req.Arguments()[0]
		server, sf, _ := req.Option("server").String()
		if sf {
			localAddr := cu.GetLocalAddr()
			n, nf, _ := req.Option("node").String()
			if nf {
				a, err := ipfsaddr.ParseString(n)
				if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}
				taddr, err := manet.ToNetAddr(a.Transport())
				if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}

				localAddr = &cu.ExternalAddr{a, taddr.(*net.TCPAddr)}
			}
			log.Debugf("Address: %s", localAddr.String())

			_, err = thrift.RunClientClosure(server, func(c *thrift.ThriftClient) (interface{}, error) {
				return nil, c.SendVerificationQuery(req.Context(), id, &casperproto.NodeInfo{
					IpfsAddr:   localAddr.IPFS().String(),
					ThriftAddr: localAddr.Thrift().String(),
				})
			})
			if err != nil {
				log.Error(err)
				res.SetError(err, cmds.ErrNormal)
				return
			}
		} else {
			err := val.PerformValidation(req.Context(), n, id)
			if err != nil {
				log.Error(err)
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}
		res.SetOutput(nil)
	},
}
