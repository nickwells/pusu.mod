package pusuclt

import (
	"time"

	"github.com/nickwells/pusu.mod/pusu"
)

// ConnInfo encapsulates the details needed to establish a connection to a
// publish/subscribe server.
//
// See [github.com/nickwells/pusuparams.mod/pusuparams] for how to provide
// collections of parameters that can be used to set these values.
type ConnInfo struct {
	SvrAddress  string        // the network address for the pub/sub server
	CertInfo    pusu.CertInfo // certificate information for the connection
	ConnTimeout time.Duration // the connection dialler timeout

	PingInterval time.Duration       // how long to wait between Pings
	pingHandler  func(time.Duration) // a func to handle ping messages
}

// NewConnInfo returns a default ConnInfo
//
// The ping handler is a function to be called with the ping round-trip time,
// pass nil to ignore ping times and suppress the pinging of the
// server. Otherwise this will be called every PingInterval.
func NewConnInfo(pingHandler func(time.Duration)) *ConnInfo {
	const (
		dfltConnTimeoutSecs  = 5
		dfltPingIntervalSecs = 2
	)

	return &ConnInfo{
		ConnTimeout:  dfltConnTimeoutSecs * time.Second,
		PingInterval: dfltPingIntervalSecs * time.Second,
		pingHandler:  pingHandler,
	}
}
