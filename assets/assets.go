//go:generate go-bindata -pkg=assets -prefix=$GOPATH/src/gx/ipfs/QmdZ4PvPHFQVLLEve7DgoKDcSY19wwpGBB1GKjjKi2rEL1 init-doc $GOPATH/src/gx/ipfs/QmdZ4PvPHFQVLLEve7DgoKDcSY19wwpGBB1GKjjKi2rEL1/dir-index-html
//go:generate gofmt -w bindata.go

package assets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	uid "github.com/Casper-dev/Casper-server/casper/uuid"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/core/coreunix"
	uio "github.com/Casper-dev/Casper-server/unixfs/io"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"

	// this import keeps gx from thinking the dep isn't used
	_ "gx/ipfs/QmdZ4PvPHFQVLLEve7DgoKDcSY19wwpGBB1GKjjKi2rEL1/dir-index-html"
)

var log = logging.Logger("assets")

// initDocPaths lists the paths for the docs we want to seed during --init
var initDocPaths = []string{
	filepath.Join("init-doc", "about"),
	filepath.Join("init-doc", "readme"),
	filepath.Join("init-doc", "help"),
	filepath.Join("init-doc", "contact"),
	filepath.Join("init-doc", "security-notes"),
	filepath.Join("init-doc", "quick-start"),
	filepath.Join("init-doc", "ping"),
}

// SeedInitDocs adds the list of embedded init documentation to the passed node, pins it and returns the root key
func SeedInitDocs(nd *core.IpfsNode) (*cid.Cid, error) {
	return addAssetList(nd, initDocPaths)
}

var initDirPath = filepath.Join(os.Getenv("GOPATH"), "gx", "ipfs", "QmdZ4PvPHFQVLLEve7DgoKDcSY19wwpGBB1GKjjKi2rEL1", "dir-index-html")
var initDirIndex = []string{
	filepath.Join(initDirPath, "knownIcons.txt"),
	filepath.Join(initDirPath, "dir-index.html"),
}

func SeedInitDirIndex(nd *core.IpfsNode) (*cid.Cid, error) {
	return addAssetList(nd, initDirIndex)
}

func addAssetList(nd *core.IpfsNode, l []string) (*cid.Cid, error) {
	dirb := uio.NewDirectory(nd.DAG)

	log.Debugf("%v", l)
	for _, p := range l {
		d, err := Asset(p)
		if err != nil {
			return nil, fmt.Errorf("assets: could load Asset '%s': %s", p, err)
		}

		log.Debugf("Few first bytes: %x", d[:uid.UUIDLen])
		// s, err := coreunix.Add(nd, bytes.NewBuffer(append(bl.NullUUID, d...)))
		s, err := coreunix.Add(nd, bytes.NewBuffer(d))
		if err != nil {
			return nil, fmt.Errorf("assets: could not Add '%s': %s", p, err)
		}

		fname := filepath.Base(p)

		c, err := cid.Decode(s)
		if err != nil {
			return nil, err
		}

		log.Debugf("Trying to find CID %s", c.String())
		node, err := nd.DAG.Get(nd.Context(), c)
		if err != nil {
			return nil, err
		}

		if err := dirb.AddChild(nd.Context(), fname, node); err != nil {
			return nil, fmt.Errorf("assets: could not add '%s' as a child: %s", fname, err)
		}
	}

	dir, err := dirb.GetNode()
	if err != nil {
		return nil, err
	}

	dcid, err := nd.DAG.Add(dir)
	if err != nil {
		return nil, fmt.Errorf("assets: DAG.Add(dir) failed: %s", err)
	}

	if err := nd.Pinning.Pin(nd.Context(), dir, true); err != nil {
		return nil, fmt.Errorf("assets: Pinning on init-docu failed: %s", err)
	}

	if err := nd.Pinning.Flush(); err != nil {
		return nil, fmt.Errorf("assets: Pinning flush failed: %s", err)
	}

	return dcid, nil
}
