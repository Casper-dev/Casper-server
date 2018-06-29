package proxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
)

var log = logging.Logger("proxy")

var isUp = false
var mu = &sync.Mutex{}
var creds = make(map[string]string)

func GetProxy(username string, password string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	creds[username] = password
	if !isUp {
		initProxy()
	}
	return fmt.Sprintf("%s::%s", username, password), nil
}

func initProxy() {
	proxy := goproxy.NewProxyHttpServer()
	auth.ProxyBasic(proxy, "Auth", func(user, passwd string) bool {
		if creds[user] == passwd {
			return true
		}
		log.Info(time.Now(), "Auth failed", user, passwd)
		return false
	})

	///TODO: verbose if linker debug value is set
	//proxy.Verbose = true
	isUp = true
	go func() {
		defer func() { isUp = false }()
		log.Error(http.ListenAndServe(":8080", proxy))
	}()
}

func GenProxyCreds(seed int64) (string, string) {
	rand.Seed(seed)
	var user, password string
	for i := 0; i < 20; i++ {
		user += string(base58.BTCAlphabet[rand.Intn(len(base58.BTCAlphabet))])
		password += string(base58.BTCAlphabet[rand.Intn(len(base58.BTCAlphabet))])
	}
	log.Info(user, password)
	return user, password
}
