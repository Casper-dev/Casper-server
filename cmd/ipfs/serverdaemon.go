package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/Casper-dev/Casper-server/core"
	"github.com/Casper-dev/Casper-SC/casper_sc"
	"github.com/Casper-dev/Casper-server/casper/casper_utils"
	"github.com/Casper-dev/Casper-SC/casperproto"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/Casper-dev/Casper-server/repo/fsrepo"

	"git.apache.org/thrift.git/lib/go/thrift"

	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"math/big"
	"github.com/Casper-dev/Casper-SC/casper"
)

var currentNode *core.IpfsNode

func serve(configRoot string) int {
	fmt.Println("serving")
	///framed := flag.Bool("framed", false, "Use framed transport")
	///buffered := flag.Bool("buffered", false, "Use buffered transport")
	///TODO: to use a custom port we need to find way for other providers to get that port
	var thriftIP string = "0.0.0.0"

	if thriftIP == "" && configRoot != "" {
		cfg, err := fsrepo.ConfigAt(configRoot)
		if err != nil {
			panic(err)
		}

		// TODO: Find out if API can be not IP4 or thrift can use IP6
		// may be add new option in config for thrift?
		thriftIP, _ = ma.StringCast(cfg.Addresses.API).ValueForProtocol(ma.P_IP4)
	}

	if thriftIP == "" {
		thriftIP = Casper_SC.GetIPv4()
	}

	addr := flag.String("addr", thriftIP+":9090", "Address to listen to")
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

	//time.Sleep(time.Duration(60-time.Now().Minute()) * time.Second)
	//fmt.Println(time.Duration(60-time.Now().Minute()) * time.Second)
	time.Sleep(time.Duration(30) * time.Second)
	fmt.Println(time.Duration(30) * time.Second)

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

		rer := regexp.MustCompile("(/ip4/[/\\d\\w.]+)")
		ipRet := strings.TrimSpace(Casper_SC.ValidateMineTX(getPingResult, client, auth))
		fmt.Println("waiting for event")
		//retString := <-sink
		//fmt.Println("event ret ", retString.Val)

		re := regexp.MustCompile("/.+?/(.+?)/")
		fmt.Println("got ip !!", ipRet, "!!")
		ipRet = rer.FindString(ipRet)
		fmt.Println("got ip !!", ipRet, "!!")
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
			resultData := Casper_SC.ValidateMineTX(sendPingResultClosure, client, auth)

			///TODO: change to working regex
			re := regexp.MustCompile("Banned!")
			fmt.Println("res data ", resultData)
			if re.FindString(resultData) != "" {
				fmt.Println("Go go replication~!")
				go startReplication(ipRet, casperclient)
			}
		}

		fmt.Println(time.Duration(10+rand.Int()%20) * time.Second)
		time.Sleep(time.Duration(10+rand.Int()%20) * time.Second)
		///fmt.Println(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Second)
		///time.Sleep(time.Duration(60-time.Now().Minute()+(rand.Int()%59)) * time.Second)
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

func startReplication(ip string, casperclient *casper.Casper) {

	// We get number of banned provider files
	n, _ := casperclient.GetNumberOfFiles(nil, ip)

	for i := int64(0); i < n.Int64(); i++ {

		// For each file we get its hash and size
		hash, size, err := casperclient.GetFile(nil, ip, big.NewInt(i))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("hs", hash, size)
		// Search for a new peer to store the file
		go replicateFile(hash, size, casperclient)
	}
}

func FilterIP(addr string) (ret string) {
	re := regexp.MustCompile("/.+?/(.+?)/")
	if len(re.FindStringSubmatch(addr)) > 1 {
		ret = re.FindStringSubmatch(addr)[1]
		fmt.Println("Getting " + re.FindStringSubmatch(addr)[1])
	} else {
		fmt.Println("Wrong ip")
	}
	return
}
func replicateFile(hash string, size *big.Int, casperclient *casper.Casper) (ret string) {
	defer func() {
		if re := recover(); re != nil {
			fmt.Println(re)
		}
	}()

	tx, err := casperclient.GetPeers(nil, size)
	if err != nil {
		fmt.Println(err)
	}
	// GetPeers returns 4 ips, but we need only one of them
	filteredIP := FilterIP(tx.Ip1)

	if filteredIP != "" {
		tclient, _ := casper_utils.RunClient(filteredIP+":9090", false)
		tclient.SendReplicationQuery(context.Background(), hash, filteredIP+":9090", size.Int64())
		casper_utils.CloseClient()
	}
	return
}
