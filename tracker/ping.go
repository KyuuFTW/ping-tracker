package tracker

import (
	"net"
	"time"
)

const (
	pingTimeout = 2 * time.Second
	pingCount   = 3
)

// MeasurePing measures TCP-based latency to a remote address by attempting
// a TCP connect. This works without raw sockets (no root needed for ICMP
// alternative). Returns average RTT and loss percentage.
func MeasurePing(addr string, port int) (rtt time.Duration, loss float64) {
	if addr == "0.0.0.0" || addr == "::" || addr == "127.0.0.1" || addr == "::1" {
		return 0, 0
	}

	target := net.JoinHostPort(addr, itoa(port))

	var totalRTT time.Duration
	var successful int

	for i := 0; i < pingCount; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target, pingTimeout)
		elapsed := time.Since(start)

		if err == nil {
			conn.Close()
			totalRTT += elapsed
			successful++
		}
	}

	if successful == 0 {
		return 0, 100.0
	}

	avgRTT := totalRTT / time.Duration(successful)
	lossPercent := float64(pingCount-successful) / float64(pingCount) * 100.0

	return avgRTT, lossPercent
}

func itoa(i int) string {
	return net.JoinHostPort("", "")[0:0] + intToStr(i)
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
