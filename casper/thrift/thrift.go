package thrift

import (
	"crypto/tls"
	"fmt"
	"time"

	"gitlab.com/casperDev/Casper-thrift/casperproto"

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

const thriftDefaultTimeout = 30 * time.Second

type ClientFunc func(*ThriftClient) (interface{}, error)

func RunClientClosure(addr string, cb ClientFunc) (interface{}, error) {
	opts := defaultThriftOpts

	var transport thrift.TTransport
	var err error
	if opts.Secure {
		cfg := &tls.Config{InsecureSkipVerify: true}
		transport, err = thrift.NewTSSLSocketTimeout(addr, cfg, opts.Timeout)
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

	return cb(&ThriftClient{CasperServerClient: casperproto.NewCasperServerClientFactory(transport, opts.ProtocolFactory)})
}

func RunClientDefault(addr string) (*casperproto.CasperServerClient, thrift.TTransport, error) {
	return RunClient(addr, defaultThriftOpts)
}

func RunClient(addr string, opts ThriftOpts) (client *casperproto.CasperServerClient, transport thrift.TTransport, err error) {
	if opts.Secure {
		cfg := &tls.Config{InsecureSkipVerify: true}
		transport, err = thrift.NewTSSLSocketTimeout(addr, cfg, opts.Timeout)
	} else {
		transport, err = thrift.NewTSocketTimeout(addr, opts.Timeout)
	}
	if err != nil {
		return nil, nil, err
	}

	transport, err = opts.TransportFactory.GetTransport(transport)
	if err != nil {
		return nil, nil, err
	}

	err = transport.Open()
	if err != nil {
		return nil, nil, err
	}
	return casperproto.NewCasperServerClientFactory(transport, opts.ProtocolFactory), transport, nil
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
