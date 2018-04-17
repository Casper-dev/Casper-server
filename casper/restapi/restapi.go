package restapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	blockservice "gitlab.com/casperDev/Casper-server/blockservice"
	uuid "gitlab.com/casperDev/Casper-server/casper/uuid"
	cmds "gitlab.com/casperDev/Casper-server/commands"
	files "gitlab.com/casperDev/Casper-server/commands/files"
	cmdsHttp "gitlab.com/casperDev/Casper-server/commands/http"
	core "gitlab.com/casperDev/Casper-server/core"
	coreCmds "gitlab.com/casperDev/Casper-server/core/commands"
	corehttp "gitlab.com/casperDev/Casper-server/core/corehttp"
	coreunix "gitlab.com/casperDev/Casper-server/core/coreunix"
	offline "gitlab.com/casperDev/Casper-server/exchange/offline"
	dag "gitlab.com/casperDev/Casper-server/merkledag"
	path "gitlab.com/casperDev/Casper-server/path"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	b58 "gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
)

const (
	CasperApiPath       = "/casper/v0"
	CasperApiFile       = "file"
	CasperApiShare      = "share"
	CasperApiStat       = "stat"
	contentTypeHeader   = "Content-Type"
	streamHeader        = "X-Stream-Output"
	contentLengthHeader = "X-Content-Length"
	linkExpireTimeout   = 5 * time.Minute
)

var mimeTypes = map[string]string{
	cmds.Protobuf: "application/protobuf",
	cmds.JSON:     "application/json",
	cmds.XML:      "application/xml",
	cmds.Text:     "text/plain",
}

var log = logging.Logger("csp/api")

func NewHandler(cctx cmds.Context, root *cmds.Command) http.Handler {
	// setup request logger
	cctx.ReqLog = new(cmds.ReqLog)

	return &handler{
		cctx: cctx,
		root: root,
	}
}

type handler struct {
	cctx cmds.Context
	root *cmds.Command
}

type commandOpts struct {
	cmdPath []string
	opts    map[string]interface{}
	args    []string
	file    files.File
	reader  func(cmds.Response) (io.Reader, error)
}

func recoverHandler() {
	if r := recover(); r != nil {
		log.Error("a panic has occurred in the commands handler!")
		log.Error(r)

		debug.PrintStack()
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer recoverHandler()

	pth := path.SplitList(strings.TrimPrefix(req.URL.Path, CasperApiPath+"/"))
	if len(pth) > 0 {
		switch pth[0] {
		case CasperApiFile:
			h.processFile(w, req)
			return
		case CasperApiShare:
			h.processShare(w, req)
			return
		}
	}
	http.Error(w, "", http.StatusNotFound)
	return
}

func (h *handler) processFile(w http.ResponseWriter, req *http.Request) {
	cmdsReq, cmdOpts, err := h.parseRequest(req)
	if err == cmdsHttp.ErrNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(cmdsReq)

	n, err := h.cctx.GetNode()
	if err != nil {
		http.Error(w, "cant get ipfs node", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(n.Context())
	defer cancel()
	if cn, ok := w.(http.CloseNotifier); ok {
		clientGone := cn.CloseNotify()
		go func() {
			select {
			case <-clientGone:
			case <-ctx.Done():
			}
			cancel()
		}()
	}

	cmdsReq.SetInvocContext(h.cctx)
	err = cmdsReq.SetRootContext(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rlog := h.cctx.ReqLog.Add(cmdsReq)
	defer rlog.Finish()

	// call the command
	res := h.root.Call(cmdsReq)

	//b, _ := httputil.DumpRequest(req, true)
	//fmt.Printf("Request:\n%s\n", string(b))

	SendResponse(w, req, res, cmdsReq, cmdOpts)
}

func randString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i, l := 0, len(letters); i < n; i++ {
		s[i] = letters[rand.Intn(l)]
	}
	return string(s)
}

const magicLength = 8

func genMagic() (magic string) {
	ok := true
	for ok {
		magic = randString(magicLength)
		_, ok = fileLinkHandlers.Load(magic)
	}
	return magic
}

var errNotShareRequest = fmt.Errorf("not a share request")

func (h *handler) processShare(w http.ResponseWriter, req *http.Request) {
	pth := getRequestPath(req)
	if len(pth) < 2 {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	if req.Method != http.MethodPost {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	n, err := h.cctx.GetNode()
	if err != nil {
		http.Error(w, "cant get ipfs node", http.StatusInternalServerError)
		return
	}

	// We need to take name of first link because we always wrap files
	// in directory
	id := uuid.UUIDToCid(b58.Decode(pth[1]))
	bserv := blockservice.New(n.Blockstore, offline.Exchange(n.Blockstore))
	dserv := dag.NewDAGService(bserv)
	node, err := dserv.Get(req.Context(), id)
	if err != nil || node == nil || len(node.Links()) != 1 {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	magic := genMagic()
	handler := http.FileServer(http.Dir(fmt.Sprintf("/ipfs/%s", id.String())))
	share := fmt.Sprintf("%s/%s/%s", CasperFileSharePath, magic, node.Links()[0].Name)
	fileLinkHandlers.Store(magic, handler)
	time.AfterFunc(linkExpireTimeout, func() {
		log.Debugf("link '%s' has expired", share)
		fileLinkHandlers.Delete(magic)
	})

	log.Debugf("file was shared at '%s'", share)
	w.WriteHeader(http.StatusOK)
	err = cmdsHttp.FlushCopy(w, strings.NewReader(share))
	if err != nil {
		log.Error(err)
	}
}

func getRequestPath(req *http.Request) []string {
	return path.SplitList(strings.TrimPrefix(req.URL.Path, CasperApiPath+"/"))
}

func (h *handler) parseRequest(req *http.Request) (cmds.Request, *commandOpts, error) {
	if !strings.HasPrefix(req.URL.Path, CasperApiPath) {
		return nil, nil, errors.New("Unexpected path prefix")
	}

	pth := getRequestPath(req)
	for k, v := range req.URL.Query() {
		fmt.Printf("%s: %s\n", k, v)
	}

	var opts *commandOpts
	var err error
	switch req.Method {
	case http.MethodPost:
		opts, err = getAddNewFileOpts(req)
	case http.MethodGet:
		if len(pth) >= 3 && pth[2] == CasperApiStat {
			opts, err = getFileStatOpts(req)
		} else {
			opts, err = getGetFileOpts(req)
		}
	case http.MethodPut:
		opts, err = getReplaceFileOpts(req)
	case http.MethodDelete:
		opts, err = getDeleteFileOpts(req)
	default:
		err = fmt.Errorf("unsupported method")
	}

	if err != nil {
		return nil, nil, err
	}

	optDefs, err := h.root.GetOptions(opts.cmdPath)
	if err != nil {
		return nil, nil, err
	}

	// ignore error because command exists
	cmd, _ := h.root.Get(opts.cmdPath[:len(opts.cmdPath)-1])
	cmdsReq, err := cmds.NewRequest(opts.cmdPath, opts.opts, opts.args, opts.file, cmd, optDefs)
	if err != nil {
		return nil, nil, err
	}

	err = cmd.CheckArguments(cmdsReq)
	if err != nil {
		return nil, nil, err
	}

	return cmdsReq, opts, nil
}

func getAddNewFileOpts(req *http.Request) (*commandOpts, error) {
	f, err := getFile(req)
	if err != nil {
		// file argument is mandatory
		return nil, err
	}
	return &commandOpts{
		cmdPath: []string{"add"},
		opts: map[string]interface{}{
			cmds.EncLong:   cmds.JSON,
			cmds.CallerOpt: cmds.CallerOptWeb,
			"quiet":        true,
			"uuid":         b58.Encode(uuid.GenUUID()),
		},
		args: []string{},
		file: f,
		reader: func(res cmds.Response) (io.Reader, error) {
			// If command was add we must return only one response
			// TODO: another method of determining what command we were executing
			outChan, _ := res.Output().(<-chan interface{})
			var out io.Reader = bytes.NewReader([]byte{})
			for obj := range outChan {
				ao := obj.(*coreunix.AddedObject)
				log.Debugf("obj %s '%s' received from add", ao.UUID, ao.Name)
				if ao.Name == coreCmds.RootObjectName {
					if b, err := json.Marshal(ao); err == nil {
						out = bytes.NewReader(b)
					}
					break
				}
			}
			return out, nil
		},
	}, nil
}

func getReplaceFileOpts(req *http.Request) (*commandOpts, error) {
	f, err := getFile(req)
	if err != nil {
		// file argument is mandatory
		return nil, err
	}
	pth := getRequestPath(req)
	return &commandOpts{
		cmdPath: []string{"add"},
		opts: map[string]interface{}{
			cmds.EncLong:   cmds.JSON,
			cmds.CallerOpt: cmds.CallerOptWeb,
			"quiet":        true,
			"uuid":         pth[1],
		},
		args: []string{},
		file: f,
		reader: func(res cmds.Response) (io.Reader, error) {
			// If command was add we must return only one response
			// TODO: another method of determining what command we were executing
			outChan, _ := res.Output().(<-chan interface{})
			var out io.Reader = bytes.NewReader([]byte{})
			for obj := range outChan {
				ao := obj.(*coreunix.AddedObject)
				log.Debugf("obj %s '%s' received from add", ao.UUID, ao.Name)
				if ao.Name == coreCmds.RootObjectName {
					if b, err := json.Marshal(ao); err == nil {
						out = bytes.NewReader(b)
					}
					break
				}
			}
			return out, nil
		},
	}, nil
}

func getGetFileOpts(req *http.Request) (*commandOpts, error) {
	pth := getRequestPath(req)
	if len(pth) < 2 {
		return nil, fmt.Errorf("name is not specified")
	}
	a, ok := req.URL.Query()["archive"]
	archive := ok && len(a) == 1 && a[0] == "1"
	cmdPath := []string{"cat"}
	if archive {
		cmdPath = []string{"get"}
	}
	return &commandOpts{
		cmdPath: cmdPath,
		opts: map[string]interface{}{
			cmds.EncLong:   cmds.JSON,
			cmds.CallerOpt: cmds.CallerOptWeb,
		},
		args: []string{getHash(pth[1])},
	}, nil
}

func getFileStatOpts(req *http.Request) (*commandOpts, error) {
	pth := getRequestPath(req)
	return &commandOpts{
		cmdPath: []string{"dag", "stat"},
		opts: map[string]interface{}{
			cmds.EncLong:   cmds.JSON,
			cmds.CallerOpt: cmds.CallerOptWeb,
		},
		args: []string{getHash(pth[1])},
	}, nil
}

func getDeleteFileOpts(req *http.Request) (*commandOpts, error) {
	cmdPath := []string{"del"}

	pth := getRequestPath(req)
	return &commandOpts{
		cmdPath: cmdPath,
		opts: map[string]interface{}{
			cmds.EncLong:   cmds.JSON,
			cmds.CallerOpt: cmds.CallerOptWeb,
		},
		args: []string{getHash(pth[1])},
	}, nil
}

func getFile(req *http.Request) (files.File, error) {
	ct := req.Header.Get(contentTypeHeader)
	mediatype, _, _ := mime.ParseMediaType(ct)
	reader, err := req.MultipartReader()
	if err != nil {
		return nil, err
	}
	return &files.MultipartFile{
		Mediatype: mediatype,
		Reader:    reader,
	}, nil
}

func getHash(id string) string {
	if u := b58.Decode(id); len(u) == uuid.UUIDLen {
		h := uuid.UUIDToHash(u).B58String()
		return h
	}
	return id
}

func guessMimeType(res cmds.Response) (string, error) {
	// Try to guess mimeType from the encoding option
	enc, found, err := res.Request().Option(cmds.EncShort).String()
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("no encoding option set")
	}

	if m, ok := mimeTypes[enc]; ok {
		return m, nil
	}

	return mimeTypes[cmds.JSON], nil
}

func SendResponse(w http.ResponseWriter, r *http.Request, res cmds.Response, req cmds.Request, cmdOpts *commandOpts) {
	h := w.Header()
	// Expose our agent to allow identification
	//h.Set("Server", "go-ipfs/"+config.CurrentVersionNumber)

	mime, err := guessMimeType(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	// if response contains an error, write an HTTP error status code
	if e := res.Error(); e != nil {
		if e.Code == cmds.ErrClient {
			status = http.StatusBadRequest
		} else {
			status = http.StatusInternalServerError
		}
		// NOTE: The error will actually be written out by the reader below
	}

	var out io.Reader
	if cmdOpts.reader == nil {
		out, err = res.Reader()
	} else {
		out, err = cmdOpts.reader(res)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set up our potential trailer
	h.Set("Trailer", cmdsHttp.StreamErrHeader)

	if res.Length() > 0 {
		h.Set(contentLengthHeader, strconv.FormatUint(res.Length(), 10))
	}

	if _, ok := res.Output().(io.Reader); ok {
		// set streams output type to text to avoid issues with browsers rendering
		// html pages on priveleged api ports
		mime = "text/plain"
		h.Set(streamHeader, "1")
	}

	// catch-all, set to text as default
	if mime == "" {
		mime = "text/plain"
	}

	h.Set(contentTypeHeader, mime)

	// set 'allowed' headers
	//h.Set("Access-Control-Allow-Headers", AllowedExposedHeaders)
	// expose those headers
	//h.Set("Access-Control-Expose-Headers", AllowedExposedHeaders)

	if r.Method == "HEAD" { // after all the headers.
		return
	}

	w.WriteHeader(status)
	err = cmdsHttp.FlushCopy(w, out)
	if err != nil {
		log.Error("err: ", err)
		w.Header().Set(cmdsHttp.StreamErrHeader, cmdsHttp.SanitizedErrStr(err))
	}
}

func CasperOption(cctx cmds.Context) corehttp.ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		cmdHandler := NewHandler(cctx, coreCmds.Root)
		mux.Handle(CasperApiPath+"/", cmdHandler)
		return mux, nil
	}
}
