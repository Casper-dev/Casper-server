package restapi

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"gitlab.com/casperDev/Casper-server/core"
	"gitlab.com/casperDev/Casper-server/core/corehttp"
	"gitlab.com/casperDev/Casper-server/path"
)

const (
	CasperFileSharePath = "/casper/share"
)

var errLinkNotFound = errors.New("link does not exist or expired")
var fileLinkHandlers = &sync.Map{}

type fileHandler struct {
}

func (_ *fileHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer recoverHandler()

	log.Debugf("got file request: %v", req.URL.Path)
	pth := path.SplitList(strings.TrimPrefix(req.URL.Path, CasperFileSharePath+"/"))
	val, ok := fileLinkHandlers.Load(pth[0])
	if !ok || val == nil {
		http.Error(w, errLinkNotFound.Error(), http.StatusNotFound)
		return
	}
	h, ok := val.(http.Handler)
	if !ok {
		http.Error(w, errLinkNotFound.Error(), http.StatusNotFound)
		return
	}

	// Path need to be relative to shared directory
	req.URL.Path = strings.TrimPrefix(req.URL.Path, fmt.Sprintf("%s/%s", CasperFileSharePath, pth[0]))
	h.ServeHTTP(w, req)
}

func CasperFileShareOption() corehttp.ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.Handle(CasperFileSharePath+"/", &fileHandler{})
		return mux, nil
	}
}
