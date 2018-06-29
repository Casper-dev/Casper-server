package dagcmd

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Casper-dev/Casper-server/casper/uuid"
	cval "github.com/Casper-dev/Casper-server/casper/validation"
	cmds "github.com/Casper-dev/Casper-server/commands"
	dag "github.com/Casper-dev/Casper-server/merkledag"
	ft "github.com/Casper-dev/Casper-server/unixfs"

	files "github.com/Casper-dev/Casper-server/commands/files"
	coredag "github.com/Casper-dev/Casper-server/core/coredag"
	path "github.com/Casper-dev/Casper-server/path"
	pin "github.com/Casper-dev/Casper-server/pin"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

var DagCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Interact with ipld dag objects.",
		ShortDescription: `
'ipfs dag' is used for creating and manipulating dag objects.

This subcommand is currently an experimental feature, but it is intended
to deprecate and replace the existing 'ipfs object' command moving forward.
		`,
	},
	Subcommands: map[string]*cmds.Command{
		"put":      DagPutCmd,
		"get":      DagGetCmd,
		"resolve":  DagResolveCmd,
		"checksum": DagChecksumCmd,
		"stat":     DagStatCmd,
	},
}

// OutputObject is the output type of 'dag put' command
type OutputObject struct {
	Cid *cid.Cid
}

// ResolveOutput is the output type of 'dag resolve' command
type ResolveOutput struct {
	Cid     *cid.Cid
	RemPath string
}

var DagPutCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Add a dag node to ipfs.",
		ShortDescription: `
'ipfs dag put' accepts input from a file or stdin and parses it
into an object of the specified format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.FileArg("object data", true, true, "The object to put").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.StringOption("format", "f", "Format that the object will be added as.").Default("cbor"),
		cmds.StringOption("input-enc", "Format that the input object will be.").Default("json"),
		cmds.BoolOption("pin", "Pin this object when adding.").Default(false),
		cmds.StringOption("hash", "Hash function to use").Default(""),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		ienc, _, _ := req.Option("input-enc").String()
		format, _, _ := req.Option("format").String()
		hash, _, err := req.Option("hash").String()
		dopin, _, err := req.Option("pin").Bool()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		// mhType tells inputParser which hash should be used. MaxUint64 means 'use
		// default hash' (sha256 for cbor, sha1 for git..)
		mhType := uint64(math.MaxUint64)

		if hash != "" {
			var ok bool
			mhType, ok = mh.Names[hash]
			if !ok {
				res.SetError(fmt.Errorf("%s in not a valid multihash name", hash), cmds.ErrNormal)
				return
			}
		}

		outChan := make(chan interface{}, 8)
		res.SetOutput((<-chan interface{})(outChan))

		addAllAndPin := func(f files.File) error {
			cids := cid.NewSet()
			b := n.DAG.Batch()

			for {
				file, err := f.NextFile()
				if err == io.EOF {
					// Finished the list of files.
					break
				} else if err != nil {
					return err
				}

				nds, err := coredag.ParseInputs(ienc, format, file, mhType, -1)
				if err != nil {
					return err
				}
				if len(nds) == 0 {
					return fmt.Errorf("no node returned from ParseInputs")
				}

				for _, nd := range nds {
					_, err := b.Add(nd)
					if err != nil {
						return err
					}
				}

				cid := nds[0].Cid()
				cids.Add(cid)
				outChan <- &OutputObject{Cid: cid}
			}

			if err := b.Commit(); err != nil {
				return err
			}

			if dopin {
				defer n.Blockstore.PinLock().Unlock()

				cids.ForEach(func(c *cid.Cid) error {
					n.Pinning.PinWithMode(c, pin.Recursive)
					return nil
				})

				err := n.Pinning.Flush()
				if err != nil {
					return err
				}
			}

			return nil
		}

		go func() {
			defer close(outChan)
			if err := addAllAndPin(req.Files()); err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}()
	},
	Type: OutputObject{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			outChan, ok := res.Output().(<-chan interface{})
			if !ok {
				return nil, u.ErrCast()
			}

			marshal := func(v interface{}) (io.Reader, error) {
				obj, ok := v.(*OutputObject)
				if !ok {
					return nil, u.ErrCast()
				}

				return strings.NewReader(obj.Cid.String() + "\n"), nil
			}

			return &cmds.ChannelMarshaler{
				Channel:   outChan,
				Marshaler: marshal,
				Res:       res,
			}, nil
		},
	},
}

const (
	startOptionName = "start"
	stopOptionName  = "stop"
	saltOptionName  = "salt"
	uuidOptionName  = "uuid"
)

var DagChecksumCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Calculate checksum for a range of file.",
		ShortDescription: `
'ipfs dag checksum' calculates sha256-checksum for a range of a file node
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "Hash or UUID of file"),
	},
	// TODO: uint64 option
	Options: []cmds.Option{
		cmds.IntOption(startOptionName, "First byte number.").Default(0),
		cmds.IntOption(stopOptionName, "Last byte number. If 0, read till end.").Default(0),
		cmds.StringOption(saltOptionName, "Salt to add before hashing.").Default(""),
		cmds.BoolOption(uuidOptionName, "Assume that ref is UUID.").Default(false),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		start, _, _ := req.Option(startOptionName).Int()
		stop, _, _ := req.Option(stopOptionName).Int()
		salt, _, _ := req.Option(saltOptionName).String()
		isUUID, _, _ := req.Option(uuidOptionName).Bool()

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		id := req.Arguments()[0]
		if isUUID {
			id = uuid.UUIDToHash(base58.Decode(req.Arguments()[0])).B58String()
		}

		p, err := path.ParsePath(id)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, _, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		cs, err := cval.ChecksumSalt(req.Context(), obj, int64(start), int64(stop), n.Resolver.DAG, []byte(salt))
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
		}

		//client.InvokeGetFileChecksum(req.Context(), "10.10.10.1:9090", req.Arguments()[0], int64(start), int64(stop), salt)

		res.SetOutput(strings.NewReader(cs.B58String() + "\n"))
	},
}

type DagStat struct {
	Name string
	Size uint64
	UUID string
	Hash string
}

var DagStatCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Get info about file.",
		ShortDescription: `
'ipfs dag stat' returns file information, such as size, name etc.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "Hash or UUID of file"),
	},
	// TODO: uint64 option
	Options: []cmds.Option{
		cmds.BoolOption(uuidOptionName, "Assume that ref is UUID.").Default(false),
	},
	Type: &DagStat{},
	Run: func(req cmds.Request, res cmds.Response) {
		isUUID, _, _ := req.Option(uuidOptionName).Bool()

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		id := req.Arguments()[0]
		if isUUID {
			id = uuid.UUIDToHash(base58.Decode(req.Arguments()[0])).B58String()
		}

		p, err := path.ParsePath(id)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, _, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		stat := &DagStat{}
		switch v := obj.(type) {
		case *dag.ProtoNode:
			fsn, err := ft.FSNodeFromBytes(v.Data())
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			if fsn.Type == ft.TDirectory && len(fsn.Data) == 0 && len(v.Links()) == 1 {
				// this directory is wrapped over one file
				// return slice of that file
				stat.Name = v.Links()[0].Name
				stat.UUID = base58.Encode(v.UUID())
				stat.Hash = v.Cid().String()

				child, err := v.Links()[0].GetNode(req.Context(), n.DAG)
				if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}

				switch c := child.(type) {
				case *dag.ProtoNode:
					fsn, err := ft.FSNodeFromBytes(c.Data())
					if err != nil {
						res.SetError(err, cmds.ErrNormal)
						return
					}
					stat.Size = fsn.FileSize()
				case *dag.RawNode:
					s, err := c.Stat()
					if err != nil {
						res.SetError(err, cmds.ErrNormal)
						return
					}
					stat.Size = uint64(s.DataSize)
				default:
					res.SetError(err, cmds.ErrNormal)
					return
				}
			}
		default:
			res.SetError(fmt.Errorf("Not a DAG node."), cmds.ErrNormal)
		}

		res.SetOutput(&stat)
	},
}

var DagGetCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Get a dag node from ipfs.",
		ShortDescription: `
'ipfs dag get' fetches a dag node from ipfs and prints it out in the specifed
format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "The object to get").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, rem, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		var out interface{} = obj
		if len(rem) > 0 {
			final, _, err := obj.Resolve(rem)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			out = final
		}

		res.SetOutput(out)
	},
	Marshalers: cmds.MarshalerMap{
		///cmds.Text: func(res cmds.Response) (io.Reader, error) {
		///	output := res.Output().(node.Node)
		///	return bytes.NewReader(output.RawData()), nil
		///},
	},
}

// DagResolveCmd returns address of highest block within a path and a path remainder
var DagResolveCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Resolve ipld block",
		ShortDescription: `
'ipfs dag resolve' fetches a dag node from ipfs, prints it's address and remaining path.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "The path to resolve").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, rem, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(&ResolveOutput{
			Cid:     obj.Cid(),
			RemPath: path.Join(rem),
		})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			output := res.Output().(*ResolveOutput)
			buf := new(bytes.Buffer)
			p := output.Cid.String()
			if output.RemPath != "" {
				p = path.Join([]string{p, output.RemPath})
			}

			buf.WriteString(p)

			return buf, nil
		},
	},
	Type: ResolveOutput{},
}
