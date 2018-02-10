package main

import (
	"crypto/tls"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"flag"

	"github.com/Casper-dev/Casper-server/core"
	"time"
	"math/rand"
	"context"
	"regexp"
	"strings"
	"github.com/Casper-dev/Casper-SC/casper_sc"
	"github.com/Casper-dev/Casper-server/casper/casper_thrift/casperproto"
	"github.com/Casper-dev/Casper-server/casper/casper_utils"
)

var currentNode *core.IpfsNode

func serve() int {
	fmt.Print("serving")
	///framed := flag.Bool("framed", false, "Use framed transport")
	///buffered := flag.Bool("buffered", false, "Use buffered transport")
	addr := flag.String("addr", Casper_SC.GetIPv4()+":9090", "Address to listen to")
	secure := flag.Bool("secure", false, "Use tls secure transport")

	flag.Parse()

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	transportFactory := thrift.NewTBufferedTransportFactory(8192)

	///if *framed {
	///	transportFactory = thrift.NewTFramedTransportFactory(transportFactory)
	///}

	go runPinger()

	// always run server here
	if err := runServer(transportFactory, protocolFactory, *addr, *secure); err != nil {
		fmt.Println("error running server:", err)
	}
	return 0
}

func runServer(transportFactory thrift.TTransportFactory, protocolFactory thrift.TProtocolFactory, addr string, secure bool) error {
	var transport thrift.TServerTransport
	var err error
	if secure {
		cfg := new(tls.Config)
		if cert, err := tls.LoadX509KeyPair("server.crt", "server.key"); err == nil {
			cfg.Certificates = append(cfg.Certificates, cert)
		} else {
			return err
		}
		transport, err = thrift.NewTSSLServerSocket(addr, cfg)
	} else {
		transport, err = thrift.NewTServerSocket(addr)
	}

	if err != nil {
		return err
	}
	fmt.Printf("%T\n", transport)
	handler := NewCasperServerHandler()
	processor := casperproto.NewCasperServerProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)

	fmt.Println("Starting the simple server... on ", addr)
	return server.Serve()
}

func runPinger() {
	time.Sleep(time.Duration(60-time.Now().Minute()) * time.Minute)
	fmt.Println(time.Duration(60-time.Now().Minute()) * time.Minute)
	for {
		casperclient, client, auth := Casper_SC.GetSC()
		fmt.Println("auth nonce", auth.Nonce)
		nonce, _ := client.PendingNonceAt(context.Background(), auth.From)
		fmt.Println("pending nonce", nonce)
		//sink := make(chan *casper.CasperReturnString)
		tx, err := casperclient.GetPingTarget(auth, casper_utils.GetCasperNodeID())
		//_, err = casperclient.WatchReturnString(nil, sink)

		nonce, _ = client.PendingNonceAt(context.Background(), auth.From)
		fmt.Println("pending nonce", nonce)
		ipRet := strings.TrimSpace("1" + casper_utils.ValidateMineTX(tx, client, err))
		fmt.Println("waiting for event")
		//retString := <-sink
		//fmt.Println("event ret ", retString.Val)

		re := regexp.MustCompile("/.+?/(.+?)/")
		fmt.Println("got ip ", ipRet)
		fmt.Println(len(ipRet))
		ips := re.FindStringSubmatch(ipRet)
		if len(ips) >= 1 {
			filteredIp := ips[1]
			fmt.Println("pinging ", filteredIp)
			tclient, err := casper_utils.RunClient(filteredIp+":9090", false)
			timestamp, err := tclient.Ping(context.TODO())

			fmt.Println(timestamp)
			nonce, _ = client.PendingNonceAt(context.Background(), auth.From)
			fmt.Println("pending nonce", nonce)
			casper_utils.ValidateMineTX(tx, client, err)
		}
		fmt.Println(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Minute)
		time.Sleep(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Minute)

	}
}
