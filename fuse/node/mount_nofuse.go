// +build linux darwin freebsd netbsd openbsd
// +build nofuse

package node

import (
	"errors"

	core "github.com/Casper-dev/Casper-server/core"
)

func Mount(node *core.IpfsNode, fsdir, nsdir string) error {
	return errors.New("not compiled in")
}
