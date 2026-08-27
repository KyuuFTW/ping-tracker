package tracker

import (
	"net"
	"time"
)

const (
	pingTimeout = 750 * time.Millisecond
	pingCount   = 2
)

// MeasurePing measures network latency and packet loss to a remote address.
// It prioritizes ICMP Echo requests (ping) to measure true end-to-end network RTT
// (bypassing synthetic TCP proxies such as ProtonVPN's VPN Accelerator).
// If ICMP ping is blocked or unsupported, it falls back to TCP connect latency.
func MeasurePing(addr string, port int) (rtt time.Duration, loss float64) {
	if addr == "0.0.0.0" || addr == "::" || addr == "127.0.0.1" || addr == "::1" {
		return 0, 0
	}

	// 1. Prioritize ICMP Echo ping (true end-to-end network latency)
	rtt, loss = measureICMP(addr, pingCount, pingTimeout)
	if loss < 100.0 {
		return rtt, loss
	}

	// 2. If ICMP is completely blocked / unresponsive (100% loss) and a TCP port exists,
	// fall back to TCP connect probing.
	if port > 0 {
		return measureTCP(addr, port, pingCount, pingTimeout)
	}

	return 0, 100.0
}

func measureTCP(addr string, port int, count int, timeout time.Duration) (time.Duration, float64) {
	target := net.JoinHostPort(addr, itoa(port))

	var totalRTT time.Duration
	var successful int

	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target, timeout)
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
	lossPercent := float64(count-successful) / float64(count) * 100.0

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
