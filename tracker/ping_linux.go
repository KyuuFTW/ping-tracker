//go:build linux

package tracker

import (
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func measureICMP(addr string, count int, timeout time.Duration) (time.Duration, float64) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return 0, 100.0
	}

	if ip.To4() != nil {
		return pingICMP4Linux(ip, count, timeout)
	}
	return pingICMP6Linux(ip, count, timeout)
}

func pingICMP4Linux(ip net.IP, count int, timeout time.Duration) (time.Duration, float64) {
	// Try unprivileged UDP ICMP socket first ("udp4"), fallback to raw ("ip4:icmp")
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		c, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			return 0, 100.0 // ICMP unavailable, will fallback to TCP
		}
	}
	defer c.Close()

	var dst net.Addr
	if c.IPv4PacketConn() != nil {
		dst = &net.IPAddr{IP: ip}
	} else {
		dst = &net.UDPAddr{IP: ip}
	}

	pid := os.Getpid() & 0xffff
	var totalRTT time.Duration
	var successful int

	rb := make([]byte, 1500)
	for i := 0; i < count; i++ {
		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   pid,
				Seq:  i + 1,
				Data: []byte("PING-TRACKER"),
			},
		}
		wb, err := wm.Marshal(nil)
		if err != nil {
			continue
		}

		start := time.Now()
		if _, err := c.WriteTo(wb, dst); err != nil {
			continue
		}

		_ = c.SetReadDeadline(time.Now().Add(timeout))
		n, _, err := c.ReadFrom(rb)
		elapsed := time.Since(start)
		if err != nil {
			continue
		}

		proto := ipv4.ICMPTypeEchoReply.Protocol()
		if c.IPv4PacketConn() == nil {
			proto = 1 // ICMP
		}
		rm, err := icmp.ParseMessage(proto, rb[:n])
		if err == nil && rm.Type == ipv4.ICMPTypeEchoReply {
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

func pingICMP6Linux(ip net.IP, count int, timeout time.Duration) (time.Duration, float64) {
	c, err := icmp.ListenPacket("udp6", "::")
	if err != nil {
		c, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
		if err != nil {
			return 0, 100.0
		}
	}
	defer c.Close()

	var dst net.Addr
	if c.IPv6PacketConn() != nil {
		dst = &net.IPAddr{IP: ip}
	} else {
		dst = &net.UDPAddr{IP: ip}
	}

	pid := os.Getpid() & 0xffff
	var totalRTT time.Duration
	var successful int

	rb := make([]byte, 1500)
	for i := 0; i < count; i++ {
		wm := icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   pid,
				Seq:  i + 1,
				Data: []byte("PING-TRACKER"),
			},
		}
		wb, err := wm.Marshal(nil)
		if err != nil {
			continue
		}

		start := time.Now()
		if _, err := c.WriteTo(wb, dst); err != nil {
			continue
		}

		_ = c.SetReadDeadline(time.Now().Add(timeout))
		n, _, err := c.ReadFrom(rb)
		elapsed := time.Since(start)
		if err != nil {
			continue
		}

		rm, err := icmp.ParseMessage(58, rb[:n]) // 58 = IPv6-ICMP
		if err == nil && rm.Type == ipv6.ICMPTypeEchoReply {
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
