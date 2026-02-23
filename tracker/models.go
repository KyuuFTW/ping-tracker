package tracker

import (
	"fmt"
	"time"
)

// ConnState represents the state of a TCP connection.
type ConnState string

const (
	StateEstablished ConnState = "ESTABLISHED"
	StateListening   ConnState = "LISTEN"
	StateTimeWait    ConnState = "TIME_WAIT"
	StateCloseWait   ConnState = "CLOSE_WAIT"
	StateSynSent     ConnState = "SYN_SENT"
	StateSynRecv     ConnState = "SYN_RECV"
	StateFinWait1    ConnState = "FIN_WAIT1"
	StateFinWait2    ConnState = "FIN_WAIT2"
	StateLastAck     ConnState = "LAST_ACK"
	StateClosing     ConnState = "CLOSING"
	StateClosed      ConnState = "CLOSED"
	StateUnknown     ConnState = "UNKNOWN"
)

// Direction indicates whether a connection is inbound or outbound.
type Direction string

const (
	Inbound  Direction = "IN"
	Outbound Direction = "OUT"
)

// Connection represents a single tracked network connection.
type Connection struct {
	// Identity
	PID       int
	AppName   string
	Protocol  string // "tcp", "tcp6", "udp", "udp6"
	Direction Direction

	// Endpoints
	LocalAddr  string
	LocalPort  int
	RemoteAddr string
	RemotePort int

	// State
	State ConnState

	// Metrics
	Ping    time.Duration // RTT latency
	Loss    float64       // packet loss percentage (0-100)
	TxBytes uint64        // bytes sent (from /proc/net)
	RxBytes uint64        // bytes received
	TxRate  float64       // bytes/sec send rate
	RxRate  float64       // bytes/sec receive rate
	ConnAge time.Duration // how long the connection has existed

	// Internal bookkeeping
	FirstSeen   time.Time
	LastUpdated time.Time
	PingCount   int
	PingFailed  int

	// Previous byte counts for rate calculation
	prevTxBytes uint64
	prevRxBytes uint64
	prevTime    time.Time
}

// Key returns a unique identifier for this connection.
func (c *Connection) Key() string {
	return fmt.Sprintf("%d:%s:%s:%d->%s:%d",
		c.PID, c.Protocol, c.LocalAddr, c.LocalPort, c.RemoteAddr, c.RemotePort)
}

// BandwidthStr returns a human-readable bandwidth string.
func FormatBytes(b float64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB/s", b/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB/s", b/(1<<10))
	default:
		return fmt.Sprintf("%.0f B/s", b)
	}
}

// FormatBytesTotal returns a human-readable total bytes string.
func FormatBytesTotal(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
