package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	bstore "gitlab.com/casperDev/Casper-server/blocks/blockstore"
	"gitlab.com/casperDev/Casper-server/blockservice"
	cu "gitlab.com/casperDev/Casper-server/casper/casper_utils"
	"gitlab.com/casperDev/Casper-server/casper/uuid"
	"gitlab.com/casperDev/Casper-server/client"
	cmds "gitlab.com/casperDev/Casper-server/commands"
	"gitlab.com/casperDev/Casper-server/commands/files"
	"gitlab.com/casperDev/Casper-server/core"
	"gitlab.com/casperDev/Casper-server/core/coreunix"
	"gitlab.com/casperDev/Casper-server/exchange/offline"
	dag "gitlab.com/casperDev/Casper-server/merkledag"
	dagtest "gitlab.com/casperDev/Casper-server/merkledag/test"
	"gitlab.com/casperDev/Casper-server/mfs"
	"gitlab.com/casperDev/Casper-server/pin"
	ft "gitlab.com/casperDev/Casper-server/unixfs"

	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmeWjRodbcZFKe5tMN7poEx3izym6osrLSnTLf9UjJZBbs/pb"
)

// Error indicating the max depth has been exceded.
var ErrDepthLimitExceeded = fmt.Errorf("depth limit exceeded")

const (
	RootObjectName    = "<root>"
	finalObjectMarker = "<end>"

	quietOptionName       = "quiet"
	quieterOptionName     = "quieter"
	silentOptionName      = "silent"
	progressOptionName    = "progress"
	trickleOptionName     = "trickle"
	wrapOptionName        = "wrap-with-directory"
	hiddenOptionName      = "hidden"
	onlyHashOptionName    = "only-hash"
	chunkerOptionName     = "chunker"
	pinOptionName         = "pin"
	rawLeavesOptionName   = "raw-leaves"
	noCopyOptionName      = "nocopy"
	fstoreCacheOptionName = "fscache"
	cidVersionOptionName  = "cid-version"
	hashOptionName        = "hash"
	uuidOptionName        = "uuid"
	passwordOptionName    = "password"
	updateOptionName      = "update"
	peersOptionName       = "peers"
	waitOptionName        = "wait"
)

const adderOutChanSize = 8

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

The wrap option, '-w', wraps the file (or files, if using the
recursive option) in a directory. This directory contains only
the files which have been added, and means that the file retains
its filename. For example:

  > ipfs add example.jpg
  added QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH example.jpg
  > ipfs add example.jpg -w
  added QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH example.jpg
  added QmaG4FuMqEBnQNn3C8XJ5bpW8kLs7zq2ZXgHptJHbKDDVx

You can now refer to the added file in a gateway, like so:

  /ipfs/QmaG4FuMqEBnQNn3C8XJ5bpW8kLs7zq2ZXgHptJHbKDDVx/example.jpg

The chunker option, '-s', specifies the chunking strategy that dictates
how to break files into blocks. Blocks with same content can
be deduplicated. The default is a fixed block size of
256 * 1024 bytes, 'size-262144'. Alternatively, you can use the
rabin chunker for content defined chunking by specifying
rabin-[min]-[avg]-[max] (where min/avg/max refer to the resulting
chunk sizes). Using other chunking strategies will produce
different hashes for the same file.

  > ipfs add --chunker=size-2048 ipfs-logo.svg
  added QmafrLBfzRLV4XSH1XcaMMeaXEUhDJjmtDfsYU95TrWG87 ipfs-logo.svg
  > ipfs add --chunker=rabin-512-1024-2048 ipfs-logo.svg
  added Qmf1hDN65tR55Ubh2RN1FPxr69xq3giVBz1KApsresY8Gn ipfs-logo.svg

You can now check what blocks have been created by:

  > ipfs object links QmafrLBfzRLV4XSH1XcaMMeaXEUhDJjmtDfsYU95TrWG87
  QmY6yj1GsermExDXoosVE3aSPxdMNYr6aKuw3nA8LoWPRS 2059
  Qmf7ZQeSxq2fJVJbCmgTrLLVN9tDR9Wy5k75DxQKuz5Gyt 1195
  > ipfs object links Qmf1hDN65tR55Ubh2RN1FPxr69xq3giVBz1KApsresY8Gn
  QmY6yj1GsermExDXoosVE3aSPxdMNYr6aKuw3nA8LoWPRS 2059
  QmerURi9k4XzKCaaPbsK6BL5pMEjF7PGphjDvkkjDtsVf3 868
  QmQB28iwSriSUSMqG2nXDTLtdPHgWb4rebBrU7Q1j4vxPv 338
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
			// if UUID is specified on client, it is an update operation
			caller, _, _ := req.Option(cmds.CallerOpt).String()
			req.SetOption(updateOptionName, caller == cmds.CallerOptClient)
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
		upd, _, _ := req.Option(updateOptionName).Bool()
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
			if upd && caller == cmds.CallerOptClient {
				newRoot, _ := pn.Copy().(*dag.ProtoNode)
				newRoot.SetUUID(uid)
				n.Pinning.PinWithMode(newRoot.Cid(), pin.Recursive)
				exch.HasBlock(newRoot)

				uid = uuid.GenUUID()
			}

			pn.SetUUID(uid)
			exch.HasBlock(pn)
			root = pn
			n.AddUUID(uuidOpt, &core.UUIDInfo{PubKey: ""})

			size, _ := pn.Size()
			log.Debug(size)
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

			var performUpdate = func(_ ma.Multiaddr) {}
			var peers []ma.Multiaddr
			if popt, pf, _ := req.Option(peersOptionName).String(); pf {
				var ps []string
				err := json.Unmarshal([]byte(popt), &ps)
				if err != nil {
					log.Error(err)
					return
				}
				for _, p := range ps {
					peers = append(peers, ma.StringCast(p))
				}
				performUpdate = func(peer ma.Multiaddr) { uploadRoot(req.Context(), n, peer, root) }
			} else if caller == cmds.CallerOptClient {
				if req.Option(updateOptionName).Found() {
					log.Debugf("UUID is specified. Existing file will be updated %s %s", root.UUID(), root.Cid().String())
					b58uuid, _, _ := req.Option(uuidOptionName).String()
					peers, err = cu.GetPeersMultiaddrsByHash(root.Cid().String())
					if err != nil {
						log.Error(err)
						return
					}
					performUpdate = func(peer ma.Multiaddr) {
						updateRoot(req.Context(), n, peer, root, b58uuid)
					}
				} else {
					size, _ := root.Size()
					peers, err = cu.GetPeersMultiaddrsBySize(int64(size))
					fmt.Println("Got peeers", peers)
					if err != nil {
						log.Error(err)
						return
					}
					performUpdate = func(peer ma.Multiaddr) {
						uploadRoot(req.Context(), n, peer, root)
					}
				}
			}

			wg := &sync.WaitGroup{}
			wg.Add(len(peers))
			for _, peer := range peers {
				go func() {
					defer wg.Done()
					performUpdate(peer)
				}()
			}
			wg.Wait()
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

func uploadRoot(ctx context.Context, n *core.IpfsNode, peer ma.Multiaddr, root *dag.ProtoNode) {
	err := n.ConnectToPeer(ctx, peer.String())
	if err != nil {
		log.Error("Failed to connect: %s", err)
		return
	}
	size, err := root.Size()
	if err != nil {
		log.Error("Cannot determine root size")
		return
	}
	addr, _ := cu.MultiaddrToTCPAddr(peer)


	thriftAddr := net.JoinHostPort(addr.IP.String(), "9090")
	err = client.HandleClientUpload(ctx, thriftAddr, root.Cid().String(), int64(size), []string{})
	if err != nil {
		log.Error("Error while uploading to %s: %s", thriftAddr, err)
	}
}

func updateRoot(ctx context.Context, n *core.IpfsNode, peer ma.Multiaddr, root *dag.ProtoNode, uuid string) {
	err := n.ConnectToPeer(ctx, peer.String())
	if err != nil {
		log.Error("Failed to connect: %s", err)
		return
	}
	size, err := root.Size()
	if err != nil {
		log.Error("Cannot determine root size")
		return
	}
	addr, _ := cu.MultiaddrToTCPAddr(peer)
	thriftAddr := net.JoinHostPort(addr.IP.String(), "9090")
	err = client.HandleClientUpdate(ctx, thriftAddr, uuid, root.Cid().String(), int64(size))
	if err != nil {
		log.Error("Error while updating on %s: %s", peer, err)
	}
}
