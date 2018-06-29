package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	bstore "github.com/Casper-dev/Casper-server/blocks/blockstore"
	"github.com/Casper-dev/Casper-server/blockservice"
	cu "github.com/Casper-dev/Casper-server/casper/casper_utils"
	sc "github.com/Casper-dev/Casper-server/casper/sc"
	"github.com/Casper-dev/Casper-server/casper/uuid"
	cmds "github.com/Casper-dev/Casper-server/commands"
	"github.com/Casper-dev/Casper-server/commands/files"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/core/coreunix"
	"github.com/Casper-dev/Casper-server/exchange/offline"
	dag "github.com/Casper-dev/Casper-server/merkledag"
	dagtest "github.com/Casper-dev/Casper-server/merkledag/test"
	"github.com/Casper-dev/Casper-server/mfs"
	ft "github.com/Casper-dev/Casper-server/unixfs"

	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmeWjRodbcZFKe5tMN7poEx3izym6osrLSnTLf9UjJZBbs/pb"
)

var AddCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Add a file or directory to ipfs.",
		ShortDescription: `
Adds contents of <path> to ipfs. Use -r to add directories (recursively).
`,
		LongDescription: `
Adds contents of <path> to ipfs. Use -r to add directories.
Note that directories are added recursively, to form the ipfs
MerkleDAG.
`,
	},

	Arguments: []cmds.Argument{
		cmds.FileArg("path", true, false, "The path to a file to be added to ipfs.").EnableRecursive().EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.OptionRecursivePath, // a builtin option that allows recursive paths (-r, --recursive)
		cmds.BoolOption(quietOptionName, "q", "Write minimal output."),
		cmds.BoolOption(quieterOptionName, "Q", "Write only final hash."),
		cmds.BoolOption(silentOptionName, "Write no output."),
		cmds.BoolOption(progressOptionName, "p", "Stream progress data."),
		cmds.BoolOption(trickleOptionName, "t", "Use trickle-dag format for dag generation."),
		cmds.BoolOption(onlyHashOptionName, "n", "Only chunk and hash - do not write to disk."),
		cmds.BoolOption(wrapOptionName, "w", "Wrap files with a directory object.").Default(true),
		cmds.BoolOption(hiddenOptionName, "H", "Include files that are hidden. Only takes effect on recursive add."),
		cmds.StringOption(chunkerOptionName, "s", "Chunking algorithm, size-[bytes] or rabin-[min]-[avg]-[max]").Default("size-262144"),
		cmds.BoolOption(pinOptionName, "Pin this object when adding.").Default(true),
		cmds.BoolOption(rawLeavesOptionName, "Use raw blocks for leaf nodes. (experimental)"),
		cmds.BoolOption(noCopyOptionName, "Add the file using filestore. (experimental)"),
		cmds.BoolOption(fstoreCacheOptionName, "Check the filestore for pre-existing blocks. (experimental)"),
		cmds.IntOption(cidVersionOptionName, "Cid version. Non-zero value will change default of 'raw-leaves' to true. (experimental)").Default(0),
		cmds.StringOption(hashOptionName, "Hash function to use. Will set Cid version to 1 if used. (experimental)").Default("sha2-256"),
		cmds.StringOption(uuidOptionName, "Base58-encoded UUID to use. Generate random by default.").Default(nil),
		cmds.BoolOption(updateOptionName, "Update file with existing UUID instead of adding new.").Default(true),
		cmds.StringOption(passwordOptionName, "Encrypt files using password (AEC-256 CTR)."),
		cmds.StringOption(peersOptionName, "JSON-encoded list of peer-multiaddrs").Default(""),
		cmds.BoolOption(waitOptionName, "Wait until file is read").Default(""),
	},
	PreRun: func(req cmds.Request) error {
		quiet, _, _ := req.Option(quietOptionName).Bool()
		quieter, _, _ := req.Option(quieterOptionName).Bool()
		quiet = quiet || quieter

		silent, _, _ := req.Option(silentOptionName).Bool()

		if quiet || silent {
			return nil
		}

		// ipfs cli progress bar defaults to true unless quiet or silent is used
		_, found, _ := req.Option(progressOptionName).Bool()
		if !found {
			req.SetOption(progressOptionName, true)
		}

		sizeFile, ok := req.Files().(files.SizeFile)
		if !ok {
			// we don't need to error, the progress bar just won't know how big the files are
			log.Warning("cannot determine size of input file")
			return nil
		}

		fmt.Println("files ", req.Files().(files.SizeFile).FullPath())

		sizeCh := make(chan int64, 1)
		req.Values()["size"] = sizeCh

		// On server we generate new UUID
		if _, uuidset, _ := req.Option(uuidOptionName).String(); uuidset {
			// if UUID is specified on client or with REST API, it is an update operation
			caller, _, _ := req.Option(cmds.CallerOpt).String()
			req.SetOption(updateOptionName, caller == cmds.CallerOptClient || caller == cmds.CallerOptWeb)
		} else {
			req.SetOption(uuidOptionName, base58.Encode(uuid.GenUUID()))
		}

		go func() {
			size, err := sizeFile.Size()
			if err != nil {
				log.Warningf("error getting files size: %s", err)
				// see comment above
				return
			}

			log.Debugf("Total size of file being added: %v\n", size)
			sizeCh <- size
		}()

		return nil
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		cfg, err := n.Repo.Config()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		// check if repo will exceed storage limit if added
		// TODO: this doesn't handle the case if the hashed file is already in blocks (deduplicated)
		// TODO: conditional GC is disabled due to it is somehow not possible to pass the size to the daemon
		//if err := corerepo.ConditionalGC(req.Context(), n, uint64(size)); err != nil {
		//	res.SetError(err, cmds.ErrNormal)
		//	return
		//}

		progress, _, _ := req.Option(progressOptionName).Bool()
		trickle, _, _ := req.Option(trickleOptionName).Bool()
		wrap, _, _ := req.Option(wrapOptionName).Bool()
		hash, _, _ := req.Option(onlyHashOptionName).Bool()
		hidden, _, _ := req.Option(hiddenOptionName).Bool()
		silent, _, _ := req.Option(silentOptionName).Bool()
		chunker, _, _ := req.Option(chunkerOptionName).String()
		dopin, _, _ := req.Option(pinOptionName).Bool()
		rawblks, rbset, _ := req.Option(rawLeavesOptionName).Bool()
		nocopy, _, _ := req.Option(noCopyOptionName).Bool()
		fscache, _, _ := req.Option(fstoreCacheOptionName).Bool()
		cidVer, _, _ := req.Option(cidVersionOptionName).Int()
		hashFunStr, hfset, _ := req.Option(hashOptionName).String()
		caller, _, _ := req.Option(cmds.CallerOpt).String()
		uuidOpt, _, _ := req.Option(uuidOptionName).String()
		//waitOpt, _, _ := req.Option(waitOptionName).Bool()

		if nocopy && !cfg.Experimental.FilestoreEnabled {
			res.SetError(errors.New("filestore is not enabled, see https://git.io/vy4XN"),
				cmds.ErrClient)
			return
		}

		if nocopy && !rbset {
			rawblks = true
		}

		if nocopy && !rawblks {
			res.SetError(fmt.Errorf("nocopy option requires '--raw-leaves' to be enabled as well"), cmds.ErrNormal)
			return
		}

		if hfset && cidVer == 0 {
			cidVer = 1
		}

		if cidVer >= 1 && !rbset {
			rawblks = true
		}

		prefix, err := dag.PrefixForCidVersion(cidVer)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		hashFunCode, ok := mh.Names[strings.ToLower(hashFunStr)]
		if !ok {
			res.SetError(fmt.Errorf("unrecognized hash function: %s", strings.ToLower(hashFunStr)), cmds.ErrNormal)
			return
		}

		prefix.MhType = hashFunCode
		prefix.MhLength = -1

		if hash {
			nilnode, err := core.NewNode(n.Context(), &core.BuildCfg{
				//TODO: need this to be true or all files
				// hashed will be stored in memory!
				NilRepo: true,
			})
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			n = nilnode
		}

		addblockstore := n.Blockstore
		if !(fscache || nocopy) {
			addblockstore = bstore.NewGCBlockstore(n.BaseBlocks, n.GCLocker)
		}

		exch := n.Exchange
		local, _, _ := req.Option("local").Bool()
		if local {
			exch = offline.Exchange(addblockstore)
		}

		bserv := blockservice.New(addblockstore, exch)
		dserv := dag.NewDAGService(bserv)

		fileAdder, err := coreunix.NewAdder(req.Context(), n.Pinning, n.Blockstore, dserv)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		outChan := make(chan interface{}, adderOutChanSize)
		res.SetOutput((<-chan interface{})(outChan))

		fileAdder.Out = outChan
		fileAdder.Chunker = chunker
		fileAdder.Progress = progress
		fileAdder.Hidden = hidden
		fileAdder.Trickle = trickle
		fileAdder.Wrap = wrap
		fileAdder.Pin = dopin
		fileAdder.Silent = silent
		fileAdder.RawLeaves = rawblks
		fileAdder.NoCopy = nocopy
		fileAdder.Prefix = &prefix

		if hash {
			md := dagtest.Mock()
			mr, err := mfs.NewRoot(req.Context(), md, ft.EmptyDirNode(), nil)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			fileAdder.SetMfsRoot(mr)
		}
		contract, err := sc.GetContract(cfg.Casper.Blockchain[cfg.Casper.UsedChain])
		if err != nil {
			if caller == cmds.CallerOptWeb {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			log.Errorf("error while receiving SC: %v", err)
		}

		var root *dag.ProtoNode
		addAllAndPin := func(f files.File) error {
			// Iterate over each top-level file and add individually. Otherwise the
			// single files.File f is treated as a directory, affecting hidden file
			// semantics.
			for {
				file, err := f.NextFile()
				if err == io.EOF {
					// Finished the list of files.
					break
				} else if err != nil {
					return err
				}

				if err := fileAdder.AddFile(file); err != nil {
					return err
				}
			}

			// copy intermediary nodes from editor to our actual dagservice
			_, err := fileAdder.Finalize()
			if err != nil {
				return err
			}

			if hash {
				return nil
			}

			// Set Root UUID as specified
			dn, _ := fileAdder.RootNode()
			pn := dn.(*dag.ProtoNode)

			uid := base58.Decode(uuidOpt)
			log.Debugf("UUID: '%s'", uuidOpt)
			pn.SetUUID(uid)
			exch.HasBlock(pn)
			root = pn
			n.AddUUID(uuidOpt, &core.UUIDInfo{PubKey: ""})

			size, _ := root.Size()
			log.Debug(size)

			if caller == cmds.CallerOptWeb {
				contract.ConfirmUpload(cu.GetLocalAddr().NodeHash(), root.Cid().String(), int64(size))
			}

			outChan <- &coreunix.AddedObject{
				Name: RootObjectName,
				Hash: pn.Cid().String(),
				UUID: base58.Encode(pn.UUID()),
				Size: strconv.FormatUint(size, 10),
			}

			return fileAdder.PinRootUUID(uid)
		}

		go func() {
			defer close(outChan)
			if err := addAllAndPin(req.Files()); err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			outChan <- &coreunix.AddedObject{
				Name: "",
				Hash: finalObjectMarker,
				UUID: "",
			}

			var peers []ma.Multiaddr
			if popt, pf, _ := req.Option(peersOptionName).String(); pf {
				var ps []string
				err := json.Unmarshal([]byte(popt), &ps)
				if err != nil {
					log.Error(err)
				} else {
					for _, p := range ps {
						peers = append(peers, ma.StringCast(p))
					}
				}

				wg := &sync.WaitGroup{}
				wg.Add(len(peers))
				for _, peer := range peers {
					go func(p ma.Multiaddr) {
						defer wg.Done()
						uploadRoot(context.Background(), n, p, root)
					}(peer)
				}
				wg.Wait()
			}
		}()
	},
	PostRun: func(req cmds.Request, res cmds.Response) {
		if res.Error() != nil {
			return
		}
		outChan, ok := res.Output().(<-chan interface{})
		if !ok {
			res.SetError(u.ErrCast(), cmds.ErrNormal)
			return
		}
		res.SetOutput(nil)

		quiet, _, _ := req.Option(quietOptionName).Bool()
		quieter, _, _ := req.Option(quieterOptionName).Bool()
		quiet = quiet || quieter

		progress, _, _ := req.Option(progressOptionName).Bool()

		var bar *pb.ProgressBar
		if progress {
			bar = pb.New64(0).SetUnits(pb.U_BYTES)
			bar.ManualUpdate = true
			bar.ShowTimeLeft = false
			bar.ShowPercent = false
			bar.Output = res.Stderr()
			bar.Start()
		}

		var sizeChan chan int64
		s, found := req.Values()["size"]
		if found {
			sizeChan = s.(chan int64)
		}

		lastFile := ""
		lastHash := ""
		var totalProgress, prevFiles, lastBytes int64

	LOOP:
		for {
			select {
			case out, ok := <-outChan:
				if !ok {
					log.Debugf("Channel is closed")
					if quieter {
						fmt.Fprintln(res.Stdout(), lastHash)
					}
					break LOOP
				}
				output := out.(*coreunix.AddedObject)
				if output.Hash == finalObjectMarker {
					fmt.Println("Waiting for replication to finish...")
					break
				}

				if len(output.Hash) > 0 {
					lastHash = output.Hash

					if !quieter {
						if progress {
							// clear progress bar line before we print "added x" output
							fmt.Fprintf(res.Stderr(), "\033[2K\r")
						}

						if quiet {
							fmt.Fprintf(res.Stdout(), "%s\n", output.Hash)
						} else {
							fmt.Fprintf(res.Stdout(), "added %s %s %s %s\n", output.Hash, output.Name, output.UUID, output.Size)
						}
					}
				} else {
					log.Debugf("add progress: %v %v\n", output.Name, output.Bytes)

					if !progress {
						continue
					}

					if len(lastFile) == 0 {
						lastFile = output.Name
					}
					if output.Name != lastFile || output.Bytes < lastBytes {
						prevFiles += lastBytes
						lastFile = output.Name
					}
					lastBytes = output.Bytes
					delta := prevFiles + lastBytes - totalProgress
					totalProgress = bar.Add64(delta)
				}

				if progress {
					bar.Update()
				}
			case size := <-sizeChan:
				if progress {
					bar.Total = size
					bar.ShowPercent = true
					bar.ShowBar = true
					bar.ShowTimeLeft = true
				}
			case <-req.Context().Done():
				res.SetError(req.Context().Err(), cmds.ErrNormal)
				return
			}
		}
	},
	Type: coreunix.AddedObject{},
}
