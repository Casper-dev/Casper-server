package casper_utils

import (
	"errors"
	"net"
	"strconv"

	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
)

// List of reserved IPs by wikipedia
// TODO make this constant
var reservedIPs = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
}

func IsIPReserved(addr net.IP) bool {
	for _, ip := range reservedIPs {
		_, ipnet, _ := net.ParseCIDR(ip)
		if addr.Mask(ipnet.Mask).Equal(ipnet.IP) {
			return true
		}
	}
	return false
}

func GetLocalIP() (ip string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	ip = ""
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if v.IP.To4() != nil && !v.IP.IsLoopback() {
				// if we treffen non-reserved IP, use it
				if !IsIPReserved(v.IP) {
					return v.IP.String()
				}
				ip = v.IP.String()
			}
		}
	}
	return ip
}

var ErrMultiaddrWrongFormat = errors.New("Multiaddr must be of form /ip4/<ip>/tcp/<port>(/...)?")

func MultiaddrToTCPAddr(maddr ma.Multiaddr) (*net.TCPAddr, error) {
	res := ma.Split(maddr)
	if len(res) < 2 {
		return nil, ErrMultiaddrWrongFormat
	}
	ip, err := res[0].ValueForProtocol(ma.P_IP4)
	if err != nil {
		return nil, ErrMultiaddrWrongFormat
	}
	sp, err := res[1].ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, ErrMultiaddrWrongFormat
	}
	port, err := strconv.Atoi(sp)
	if err != nil {
		return nil, ErrMultiaddrWrongFormat
	}
	return &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}, nil
}

func FilterIP(addr string) (ip string) {
	a, err := ma.NewMultiaddr(addr)
	if err != nil {
		return
	}

	ip, err = a.ValueForProtocol(ma.P_IP4)
	if err != nil {
		return
	}

	return ip
}
