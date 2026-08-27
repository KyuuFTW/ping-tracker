//go:build windows

package tracker

import (
	"encoding/binary"
	"net"
	"syscall"
	"time"
	"unsafe"
)

var (
	modIphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procIcmpCreateFile  = modIphlpapi.NewProc("IcmpCreateFile")
	procIcmpCloseHandle = modIphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = modIphlpapi.NewProc("IcmpSendEcho")
	procIcmp6CreateFile = modIphlpapi.NewProc("Icmp6CreateFile")
	procIcmp6SendEcho2  = modIphlpapi.NewProc("Icmp6SendEcho2")
)

type ipOptionInformation struct {
	Ttl         byte
	Tos         byte
	Flags       byte
	OptionsSize byte
	_           [4]byte
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

type sockaddrIn6 struct {
	Sin6Family   int16
	Sin6Port     uint16
	Sin6FlowInfo uint32
	Sin6Addr     [16]byte
	Sin6ScopeID  uint32
}

type ipv6AddressEx struct {
	Sin6Port     uint16
	Sin6FlowInfo uint16
	Sin6Addr     [16]byte
	Sin6ScopeID  uint32
}

type icmpv6EchoReply struct {
	Address       ipv6AddressEx
	Status        uint32
	RoundTripTime uint32
}

func measureICMP(addr string, count int, timeout time.Duration) (time.Duration, float64) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return 0, 100.0
	}

	if ip4 := ip.To4(); ip4 != nil {
		return pingICMP4Windows(ip4, count, timeout)
	}
	return pingICMP6Windows(ip, count, timeout)
}

func pingICMP4Windows(ip4 net.IP, count int, timeout time.Duration) (time.Duration, float64) {
	h, _, _ := procIcmpCreateFile.Call()
	if h == 0 || h == ^uintptr(0) {
		return 0, 100.0
	}
	defer procIcmpCloseHandle.Call(h)

	dest := binary.LittleEndian.Uint32(ip4.To4())
	reqData := []byte("ping-tracker")
	replyBufferSize := uint32(unsafe.Sizeof(icmpEchoReply{}) + uintptr(len(reqData)) + 64)
	replyBuffer := make([]byte, replyBufferSize)
	timeoutMS := uint32(timeout / time.Millisecond)

	var totalRTT time.Duration
	var successful int

	for i := 0; i < count; i++ {
		start := time.Now()
		ret, _, _ := procIcmpSendEcho.Call(
			h,
			uintptr(dest),
			uintptr(unsafe.Pointer(&reqData[0])),
			uintptr(len(reqData)),
			0,
			uintptr(unsafe.Pointer(&replyBuffer[0])),
			uintptr(replyBufferSize),
			uintptr(timeoutMS),
		)
		elapsed := time.Since(start)

		if ret > 0 && len(replyBuffer) >= 12 {
			status := binary.LittleEndian.Uint32(replyBuffer[4:8])
			if status == 0 { // IP_SUCCESS
				rttMs := binary.LittleEndian.Uint32(replyBuffer[8:12])
				rtt := time.Duration(rttMs) * time.Millisecond
				if rtt == 0 {
					rtt = elapsed
				}
				totalRTT += rtt
				successful++
			}
		}
	}

	if successful == 0 {
		return 0, 100.0
	}

	avgRTT := totalRTT / time.Duration(successful)
	lossPercent := float64(count-successful) / float64(count) * 100.0
	return avgRTT, lossPercent
}

func pingICMP6Windows(ip6 net.IP, count int, timeout time.Duration) (time.Duration, float64) {
	h, _, _ := procIcmp6CreateFile.Call()
	if h == 0 || h == ^uintptr(0) {
		return 0, 100.0
	}
	defer procIcmpCloseHandle.Call(h)

	var srcAddr sockaddrIn6
	srcAddr.Sin6Family = 23 // AF_INET6

	var destAddr sockaddrIn6
	destAddr.Sin6Family = 23 // AF_INET6
	copy(destAddr.Sin6Addr[:], ip6.To16())

	reqData := []byte("ping-tracker")
	replyBufferSize := uint32(unsafe.Sizeof(icmpv6EchoReply{}) + uintptr(len(reqData)) + 64)
	replyBuffer := make([]byte, replyBufferSize)
	timeoutMS := uint32(timeout / time.Millisecond)

	var totalRTT time.Duration
	var successful int

	for i := 0; i < count; i++ {
		start := time.Now()
		ret, _, _ := procIcmp6SendEcho2.Call(
			h,
			0,
			0,
			0,
			uintptr(unsafe.Pointer(&srcAddr)),
			uintptr(unsafe.Pointer(&destAddr)),
			uintptr(unsafe.Pointer(&reqData[0])),
			uintptr(len(reqData)),
			0,
			uintptr(unsafe.Pointer(&replyBuffer[0])),
			uintptr(replyBufferSize),
			uintptr(timeoutMS),
		)
		elapsed := time.Since(start)

		if ret > 0 && len(replyBuffer) >= 32 {
			status := binary.LittleEndian.Uint32(replyBuffer[24:28])
			if status == 0 {
				rttMs := binary.LittleEndian.Uint32(replyBuffer[28:32])
				rtt := time.Duration(rttMs) * time.Millisecond
				if rtt == 0 {
					rtt = elapsed
				}
				totalRTT += rtt
				successful++
			}
		}
	}

	if successful == 0 {
		return 0, 100.0
	}

	avgRTT := totalRTT / time.Duration(successful)
	lossPercent := float64(count-successful) / float64(count) * 100.0
	return avgRTT, lossPercent
}
