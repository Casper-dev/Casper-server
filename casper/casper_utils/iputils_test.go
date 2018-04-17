package casper_utils

import (
	"net"
	"testing"
)

var (
	classA = []string{"10.0.0.0", "10.255.255.255"}
	classB = []string{"172.16.0.0", "172.31.255.255"}
	classC = []string{"192.168.0.0", "192.168.255.255"}
)

func TestIsIPReserved(t *testing.T) {
	if ip := "127.0.0.1"; !IsIPReserved(net.ParseIP(ip)) {
		t.Errorf("Loopback IP '%s' is reserved", ip)
	}

	for _, ip := range classA {
		if !IsIPReserved(net.ParseIP(ip)) {
			t.Errorf("Class A IP '%s' is reserved", ip)
		}
	}
	for _, ip := range classB {
		if !IsIPReserved(net.ParseIP(ip)) {
			t.Errorf("Class B IP '%s' is reserved", ip)
		}
	}
	for _, ip := range classC {
		if !IsIPReserved(net.ParseIP(ip)) {
			t.Errorf("Class C IP '%s' is reserved", ip)
		}
	}
}
