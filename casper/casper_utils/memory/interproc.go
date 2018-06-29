package memory

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/Casper-dev/Casper-server/exchange/bitswap/decision"
)

type Hashes struct {
}

func (addr *Hashes) GetAddresses(hash *map[string]bool) error {
	*hash = decision.AllowedHashes
	return nil
}

func ServeRPC() {
	hashes := new(Hashes)
	rpc.Register(hashes)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":13524")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func GetRPC() {
	client, err := rpc.DialHTTP("tcp", ":13524")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var reply map[string]bool
	err = client.Call("Arith.GetAddresses", nil, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %s", reply)
	decision.AllowedHashes = reply
	return
}
