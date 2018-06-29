package restapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	cmds "github.com/Casper-dev/Casper-server/commands"
	cmdsHttp "github.com/Casper-dev/Casper-server/commands/http"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-server/core/corehttp"
	"github.com/Casper-dev/Casper-server/core/coreunix"
)

const (
	// CasperFileSharePath is API prefix to all file links
	CasperFileSharePath = "/casper/share"
)

var errLinkNotFound = errors.New("link does not exist or expired")
var fileLinkHandlers = &sync.Map{}

type fileHandler struct {
	cctx cmds.Context
}

func (fh *fileHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer recoverHandler()

	log.Debugf("got file request: %v", req.URL.Path)
	magic := req.URL.Path
	val, ok := fileLinkHandlers.Load(magic)
	if !ok || val == nil {
		http.Error(w, errLinkNotFound.Error(), http.StatusNotFound)
		return
	}

	h := w.Header()
	h.Set(ACAOrigin, "*")

	n, _ := fh.cctx.GetNode()
	out, size, err := cat(req.Context(), n, val.(string))
	log.Debugf("%v %d %v", out, size, err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := http.StatusOK

	mime := req.Header.Get(contentTypeHeader)
	if mime == "" {
		mime = "text/plain"
	}

	h.Set("Trailer", cmdsHttp.StreamErrHeader)
	h.Set(contentLengthHeader, strconv.FormatUint(size, 10))
	h.Set(contentTypeHeader, mime)
	h.Set(streamHeader, "1")

	w.WriteHeader(status)
	if err = cmdsHttp.FlushCopy(w, out); err != nil {
		log.Error("err: ", err)
		h.Set(cmdsHttp.StreamErrHeader, cmdsHttp.SanitizedErrStr(err))
	}
}

func cat(ctx context.Context, node *core.IpfsNode, path string) (io.Reader, uint64, error) {
	read, err := coreunix.Cat(ctx, node, path)
	if err != nil {
		return nil, 0, err
	}
	return read, read.Size(), nil
}

func CasperFileShareOption(cctx cmds.Context) corehttp.ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		p := CasperFileSharePath + "/"
		mux.Handle(p, http.StripPrefix(p, &fileHandler{cctx}))
		return mux, nil
	}
}
