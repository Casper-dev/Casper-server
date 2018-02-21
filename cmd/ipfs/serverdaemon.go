package main

import (
	"crypto/tls"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"flag"
	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-SC/casper_sc"
	"time"

	"math/rand"
	"context"
	"regexp"
	"strings"

	"github.com/Casper-dev/Casper-server/casper/casper_utils"
	"github.com/Casper-dev/Casper-SC/casperproto"
	"github.com/ethereum/go-ethereum/core/types"
)

var currentNode *core.IpfsNode

func serve() int {
	fmt.Print("serving")
	///framed := flag.Bool("framed", false, "Use framed transport")
	///buffered := flag.Bool("buffered", false, "Use buffered transport")
	///TODO: to use a custom port we need to find way for other providers to get that port
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

var thriftTimeout = 30

func runPinger() {

	time.Sleep(time.Duration(60-time.Now().Minute()) * 5 * time.Second)
	fmt.Println(time.Duration(60-time.Now().Minute()) * 5 * time.Second)

	for {
		casperclient, client, auth := Casper_SC.GetSC()
		fmt.Println("auth nonce", auth.Nonce)
		nonce, _ := client.PendingNonceAt(context.Background(), auth.From)
		fmt.Println("pending nonce", nonce)
		//sink := make(chan *casper.CasperReturnString)

		getPingResult := func() (*types.Transaction, error) {
			return casperclient.GetPingTarget(auth, casper_utils.GetCasperNodeID())
		}
		//_, err = casperclient.WatchReturnString(nil, sink)

		nonce, _ = client.PendingNonceAt(context.Background(), auth.From)
		fmt.Println("pending nonce", nonce)
		ipRet := strings.TrimSpace(Casper_SC.ValidateMineTX(getPingResult, client))
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

			timestamp := callPing(filteredIp + ":9090")
			fmt.Println(timestamp)
			success := true
			if timestamp < time.Now().Unix()-int64(thriftTimeout) {
				success = false
			}

			sendPingResultClosure := func() (*types.Transaction, error) {
				return casperclient.SendPingResult(auth, ipRet, success)
			}

			nonce, _ = client.PendingNonceAt(context.Background(), auth.From)
			fmt.Println("pending nonce", nonce)
			resultData := Casper_SC.ValidateMineTX(sendPingResultClosure, client)

			///TODO: change to working regex
			re := regexp.MustCompile("/.+?/(.+?)/")
			fmt.Println("res data ", resultData)
			if len(re.FindStringSubmatch(resultData)) > 1 {
				go startReplication(ipRet)
			}
		}

		fmt.Println(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * 5 * time.Second)
		time.Sleep(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * 5 * time.Second)
	}
}

func callPing(ip string) (timestamp int64) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	tclient, _ := casper_utils.RunClient(ip, false)
	timestamp, _ = tclient.Ping(context.Background())
	casper_utils.CloseClient()
	return
}

func startReplication(ip string) {

}
