package config

type SwarmConfig struct {
	AddrFilters             []string
	DisableBandwidthMetrics bool
	DisableNatPortMap       bool
	DisableRelay            bool
	EnableRelayHop          bool
	NAT                     NATOpts // STUN/TURN server addresses

	ConnMgr ConnMgr
}

// ConnMgr defines configuration options for the libp2p connection manager
type ConnMgr struct {
	Type        string
	LowWater    int
	HighWater   int
	GracePeriod string
}

// NATOpts defines configuration option for IP determination
// if TraversalSC is false, one of local IPs will be sent to SC
type NATOpts struct {
	TraversalSC bool
	StunServers []string `json:",omitempty"`
	// TODO add use credentials
	TurnServers []TurnServer `json:",omitempty"`
}

type TurnServer struct {
	Address  string
	User     string
	Password string
}
