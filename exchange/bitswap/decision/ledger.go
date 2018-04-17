package decision

import (
	"sync"
	"time"

	wl "gitlab.com/casperDev/Casper-server/exchange/bitswap/wantlist"

	"gitlab.com/casperDev/Casper-SC/casper_sc"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/ethereum/go-ethereum/common"
)

func newLedger(p peer.ID) *ledger {
	return &ledger{
		wantList:   wl.New(),
		Partner:    p,
		sentToPeer: make(map[string]time.Time),
	}
}

var AllowedHashes map[string][]string

func AllowHash(hash, wallet string) {
	for _, w := range AllowedHashes[hash] {
		if w == wallet {
			return
		}
	}
	AllowedHashes[hash] = append(AllowedHashes[hash], wallet)
}

func DisallowHash(hash, wallet string) {
	for i, w := range AllowedHashes[hash] {
		if w == wallet {
			if len(AllowedHashes[hash]) == 1 {
				delete(AllowedHashes, hash)
			} else {
				var tail []string
				if i+1 < len(AllowedHashes[hash]) {
					tail = AllowedHashes[hash][i+1:]
				}
				AllowedHashes[hash] = append(AllowedHashes[hash][:i], tail...)
			}
			return
		}
	}
}

// ledger stores the data exchange relationship between two peers.
// NOT threadsafe
type ledger struct {
	// Partner is the remote Peer.
	Partner peer.ID

	// Accounting tracks bytes sent and recieved.
	Accounting debtRatio

	// lastExchange is the time of the last data exchange.
	lastExchange time.Time

	// exchangeCount is the number of exchanges with this peer
	exchangeCount uint64

	// wantList is a (bounded, small) set of keys that Partner desires.
	wantList *wl.Wantlist

	// sentToPeer is a set of keys to ensure we dont send duplicate blocks
	// to a given peer
	sentToPeer map[string]time.Time

	// ref is the reference count for this ledger, its used to ensure we
	// don't drop the reference to this ledger in multi-connection scenarios
	ref int

	lk sync.Mutex
}

type Receipt struct {
	Peer      string
	Value     float64
	Sent      uint64
	Recv      uint64
	Exchanged uint64
}

type debtRatio struct {
	BytesSent uint64
	BytesRecv uint64
}

func (dr *debtRatio) Value() float64 {
	return float64(dr.BytesSent) / float64(dr.BytesRecv+1)
}

func (l *ledger) SentBytes(n int) {
	l.exchangeCount++
	l.lastExchange = time.Now()
	l.Accounting.BytesSent += uint64(n)
}

func (l *ledger) ReceivedBytes(n int) {
	l.exchangeCount++
	l.lastExchange = time.Now()
	l.Accounting.BytesRecv += uint64(n)
}

func (l *ledger) Wants(k *cid.Cid, priority int) {
	l.wantList.Add(k, priority)
	return
	log.Debugf("peer %s wants %s", l.Partner, k)
	if wallets, ok := AllowedHashes[k.String()]; ok {
		casperclient, _, _, _ := Casper_SC.GetSC()

		// TODO check add info to wantlist about wallet
		isPre, _ := casperclient.IsPrepaid(nil, common.HexToAddress(wallets[0]))
		log.Debugf("Wallet %s prepaid=%t", isPre)
		if isPre {
			l.wantList.Add(k, priority)
			log.Infof("Hash %s was allowed to download with wallet %s\n", k.String(), wallets[0])
		}
	}
}

func (l *ledger) CancelWant(k *cid.Cid) {
	l.wantList.Remove(k)
}

func (l *ledger) WantListContains(k *cid.Cid) (*wl.Entry, bool) {
	return l.wantList.Contains(k)
}

func (l *ledger) ExchangeCount() uint64 {
	return l.exchangeCount
}
