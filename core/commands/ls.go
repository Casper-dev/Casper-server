package commands

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	blockservice "gitlab.com/casperDev/Casper-server/blockservice"
	cmds "gitlab.com/casperDev/Casper-server/commands"
	core "gitlab.com/casperDev/Casper-server/core"
	offline "gitlab.com/casperDev/Casper-server/exchange/offline"
	merkledag "gitlab.com/casperDev/Casper-server/merkledag"
	path "gitlab.com/casperDev/Casper-server/path"
	unixfs "gitlab.com/casperDev/Casper-server/unixfs"
	uio "gitlab.com/casperDev/Casper-server/unixfs/io"
	unixfspb "gitlab.com/casperDev/Casper-server/unixfs/pb"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
)

type LsLink struct {
	Name, Hash string
	Size       uint64
	Type       unixfspb.Data_DataType
}

type LsObject struct {
	Hash  string
	Links []LsLink
}

type LsOutput struct {
	Objects []LsObject
}

var LsCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "List directory contents for Unix filesystem objects.",
		ShortDescription: `
Displays the contents of an IPFS or IPNS object(s) at the given path, with
the following format:

  <link base58 hash> <link size in bytes> <link name>

The JSON output contains type information.
`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, true, "The path to the IPFS object(s) to list links from.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.BoolOption("headers", "v", "Print table headers (Hash, Size, Name).").Default(false),
		cmds.BoolOption("resolve-type", "Resolve linked objects to find out their types.").Default(true),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		// get options early -> exit early in case of error
		if _, _, err := req.Option("headers").Bool(); err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		resolve, _, err := req.Option("resolve-type").Bool()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		dserv := nd.DAG
		if !resolve {
			offlineexch := offline.Exchange(nd.Blockstore)
			bserv := blockservice.New(nd.Blockstore, offlineexch)
			dserv = merkledag.NewDAGService(bserv)
		}

		paths := req.Arguments()

		var dagnodes []node.Node
		for _, fpath := range paths {
			p, err := path.ParsePath(fpath)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			r := &path.Resolver{
				DAG:         nd.DAG,
				ResolveOnce: uio.ResolveUnixfsOnce,
			}

			dagnode, err := core.Resolve(req.Context(), nd.Namesys, r, p)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			dagnodes = append(dagnodes, dagnode)
		}

		output := make([]LsObject, len(req.Arguments()))
		for i, dagnode := range dagnodes {
			dir, err := uio.NewDirectoryFromNode(nd.DAG, dagnode)
			if err != nil && err != uio.ErrNotADir {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			var links []*node.Link
			if dir == nil {
				links = dagnode.Links()
			} else {
				links, err = dir.Links(req.Context())
				if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}
			}

			output[i] = LsObject{
				Hash:  paths[i],
				Links: make([]LsLink, len(links)),
			}

			for j, link := range links {
				t := unixfspb.Data_DataType(-1)

				linkNode, err := link.GetNode(req.Context(), dserv)
				if err == merkledag.ErrNotFound && !resolve {
					// not an error
					linkNode = nil
				} else if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}

				if pn, ok := linkNode.(*merkledag.ProtoNode); ok {
					d, err := unixfs.FromBytes(pn.Data())
					if err != nil {
						res.SetError(err, cmds.ErrNormal)
						return
					}

					t = d.GetType()
				}
				output[i].Links[j] = LsLink{
					Name: link.Name,
					Hash: link.Cid.String(),
					Size: link.Size,
					Type: t,
				}
			}
		}

		res.SetOutput(&LsOutput{output})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {

			headers, _, _ := res.Request().Option("headers").Bool()
			output := res.Output().(*LsOutput)
			buf := new(bytes.Buffer)
			w := tabwriter.NewWriter(buf, 1, 2, 1, ' ', 0)
			fmt.Println(len(output.Objects))
			SizeOut = 0
			for _, object := range output.Objects {
				if len(output.Objects) > 1 {
					fmt.Fprintf(w, "%s:\n", object.Hash)
				}
				if headers {
					fmt.Fprintln(w, "Hash\tSize\tName")
				}
				for _, link := range object.Links {
					if link.Type == unixfspb.Data_Directory {
						link.Name += "/"
					}
					SizeOut += link.Size
					//fmt.Fprintf(w, "%s\t%v\t%s\n", link.Hash, link.Size, link.Name)
				}
				if len(output.Objects) > 1 {
					fmt.Fprintln(w)
				}
			}
			w.Flush()

			return buf, nil
		},
	},
	Type: LsOutput{},
}

var SizeOut uint64
