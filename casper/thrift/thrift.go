package thrift

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Casper-dev/Casper-thrift/casperproto"

	"git.apache.org/thrift.git/lib/go/thrift"
)

type ThriftOpts struct {
	TransportFactory thrift.TTransportFactory
	ProtocolFactory  thrift.TProtocolFactory
	Secure           bool
	Timeout          time.Duration
}

type ThriftClient struct {
	*casperproto.CasperServerClient
}

var defaultThriftOpts = ThriftOpts{
	ProtocolFactory:  thrift.NewTBinaryProtocolFactoryDefault(),
	TransportFactory: thrift.NewTBufferedTransportFactory(8192),
	Secure:           false,
	Timeout:          thriftDefaultTimeout,
}
var defaultHTTPProfocolFactory = thrift.NewTBinaryProtocolFactoryDefault()

const (
	CasperThriftApi      = "/casper/thrift"
	thriftDefaultTimeout = 2 * time.Minute
)

type ClientFunc func(*ThriftClient) (interface{}, error)

type handler struct {
	sh casperproto.CasperServer
	hf http.HandlerFunc
}

func NewHandler(sh casperproto.CasperServer) *handler {
	return &handler{sh, NewHandlerFunc(sh)}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	NewHandlerFunc(h.sh)(w, r)
}

func newThriftClient(t thrift.TTransport, pf thrift.TProtocolFactory) *ThriftClient {
	return &ThriftClient{CasperServerClient: casperproto.NewCasperServerClient(
		thrift.NewTStandardClient(pf.GetProtocol(t), pf.GetProtocol(t)),
	)}
}

func RunClientClosureHTTP(url string, cb ClientFunc) (interface{}, error) {
	transport, err := thrift.NewTHttpClientWithOptions(url, thrift.THttpClientOptions{&http.Client{Timeout: thriftDefaultTimeout}})
	if err != nil {
		return nil, err
	}

	err = transport.Open()
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	return cb(newThriftClient(transport, defaultHTTPProfocolFactory))
}

func NewHandlerFunc(handler casperproto.CasperServer) http.HandlerFunc {
	pf := defaultHTTPProfocolFactory
	processor := casperproto.NewCasperServerProcessor(handler)
	return func(w http.ResponseWriter, r *http.Request) {
		t := thrift.NewStreamTransport(r.Body, w)
		processor.Process(context.TODO(), pf.GetProtocol(t), pf.GetProtocol(t))
	}
}

func RunClientClosure(addr string, cb ClientFunc) (interface{}, error) {
	return RunClientClosureOpts(addr, cb, defaultThriftOpts)
}

func RunClientClosureOpts(addr string, cb ClientFunc, opts ThriftOpts) (result interface{}, err error) {
	var transport thrift.TTransport
	if opts.Secure {
		transport, err = thrift.NewTSSLSocketTimeout(addr, &tls.Config{InsecureSkipVerify: true}, opts.Timeout)
	} else {
		transport, err = thrift.NewTSocketTimeout(addr, opts.Timeout)
	}
	if err != nil {
		return nil, err
	}

	transport, err = opts.TransportFactory.GetTransport(transport)
	if err != nil {
		return nil, err
	}

	err = transport.Open()
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	return cb(newThriftClient(transport, opts.ProtocolFactory))
}

func RunServerDefault(addr string, handler casperproto.CasperServer) error {
	return RunServer(addr, handler, defaultThriftOpts)
}

func RunServer(addr string, handler casperproto.CasperServer, opts ThriftOpts) error {
	var transport thrift.TServerTransport
	var err error
	if opts.Secure {
		cfg := new(tls.Config)
		if cert, err := tls.LoadX509KeyPair("server.crt", "server.key"); err == nil {
			cfg.Certificates = append(cfg.Certificates, cert)
		} else {
			return err
		}
		transport, err = thrift.NewTSSLServerSocketTimeout(addr, cfg, opts.Timeout)
	} else {
		transport, err = thrift.NewTServerSocketTimeout(addr, opts.Timeout)
	}
	if err != nil {
		return err
	}

	processor := casperproto.NewCasperServerProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, transport, opts.TransportFactory, opts.ProtocolFactory)

	fmt.Printf("Starting the simple server on %s ...\n", addr)

	return server.Serve()
}
