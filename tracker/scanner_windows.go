//go:build windows

package tracker

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modiphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modpsapi    = syscall.NewLazyDLL("psapi.dll")

	procGetExtendedTcpTable      = modiphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable      = modiphlpapi.NewProc("GetExtendedUdpTable")
	procOpenProcess              = modkernel32.NewProc("OpenProcess")
	procCloseHandle              = modkernel32.NewProc("CloseHandle")
	procGetProcessImageFileNameW = modkernel32.NewProc("QueryFullProcessImageNameW")
)

const (
	TCP_TABLE_OWNER_PID_ALL = 5
	UDP_TABLE_OWNER_PID     = 1
	AF_INET                 = 2
	AF_INET6                = 23

	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
)

// Windows TCP connection states
var winTCPStates = map[uint32]ConnState{
	1:  StateClosed,
	2:  StateListening,
	3:  StateSynSent,
	4:  StateSynRecv,
	5:  StateEstablished,
	6:  StateFinWait1,
	7:  StateFinWait2,
	8:  StateCloseWait,
	9:  StateClosing,
	10: StateLastAck,
	11: StateTimeWait,
	12: StateClosed, // MIB_TCP_STATE_DELETE_TCB
}

// MIB_TCPROW_OWNER_PID structure
type tcpRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

// MIB_UDPROW_OWNER_PID structure
type udpRowOwnerPID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPid uint32
}

// MIB_TCP6ROW_OWNER_PID structure
type tcp6RowOwnerPID struct {
	LocalAddr     [16]byte
	LocalScopeId  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeId uint32
	RemotePort    uint32
	State         uint32
	OwningPid     uint32
}

// MIB_UDP6ROW_OWNER_PID structure
type udp6RowOwnerPID struct {
	LocalAddr    [16]byte
	LocalScopeId uint32
	LocalPort    uint32
	OwningPid    uint32
}

// ScanConnections uses Windows API to discover active connections.
func ScanConnections() ([]*Connection, error) {
	now := time.Now()

	var conns []*Connection

	// TCP IPv4
	if entries, err := getTCPTable(); err == nil {
		for _, e := range entries {
			conns = append(conns, e.toConnection(now))
		}
	}

	// TCP IPv6
	if entries, err := getTCP6Table(); err == nil {
		for _, e := range entries {
			conns = append(conns, e.toConnection(now))
		}
	}

	// UDP IPv4
	if entries, err := getUDPTable(); err == nil {
		for _, e := range entries {
			conns = append(conns, e.toConnection(now))
		}
	}

	// UDP IPv6
	if entries, err := getUDP6Table(); err == nil {
		for _, e := range entries {
			conns = append(conns, e.toConnection(now))
		}
	}

	return conns, nil
}

// connEntry is a unified entry before converting to Connection
type connEntry struct {
	protocol   string
	localAddr  string
	localPort  int
	remoteAddr string
	remotePort int
	state      ConnState
	pid        int
}

func (e *connEntry) toConnection(now time.Time) *Connection {
	name := getProcessName(e.pid)
	if name == "" {
		name = "unknown"
	}

	dir := Outbound
	if e.state == StateListening || e.remoteAddr == "0.0.0.0" || e.remoteAddr == "::" {
		dir = Inbound
	}

	return &Connection{
		PID:         e.pid,
		AppName:     name,
		Protocol:    e.protocol,
		Direction:   dir,
		LocalAddr:   e.localAddr,
		LocalPort:   e.localPort,
		RemoteAddr:  e.remoteAddr,
		RemotePort:  e.remotePort,
		State:       e.state,
		FirstSeen:   now,
		LastUpdated: now,
	}
}

// networkToHostPort converts a network-byte-order port (uint32 with port in high 16 bits) to host port.
func networkToHostPort(p uint32) int {
	// The port is stored in network byte order in the first 2 bytes of the uint32
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, p)
	return int(binary.BigEndian.Uint16(b[:2]))
}

// uint32ToIP converts a uint32 to an IPv4 string.
func uint32ToIP(addr uint32) string {
	ip := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ip, addr)
	return ip.String()
}

// getTCPTable retrieves TCP IPv4 connections.
func getTCPTable() ([]connEntry, error) {
	var size uint32
	// First call to get size
	ret, _, _ := procGetExtendedTcpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0, // no sort
		AF_INET,
		TCP_TABLE_OWNER_PID_ALL,
		0,
	)
	if ret != 0 && ret != 122 { // 122 = ERROR_INSUFFICIENT_BUFFER
		return nil, fmt.Errorf("GetExtendedTcpTable size query failed: %d", ret)
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET,
		TCP_TABLE_OWNER_PID_ALL,
		0,
	)
	if ret != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable failed: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	var entries []connEntry

	rowSize := unsafe.Sizeof(tcpRowOwnerPID{})
	for i := uint32(0); i < numEntries; i++ {
		row := (*tcpRowOwnerPID)(unsafe.Pointer(&buf[4+uintptr(i)*rowSize]))

		state, ok := winTCPStates[row.State]
		if !ok {
			state = StateUnknown
		}

		entries = append(entries, connEntry{
			protocol:   "tcp",
			localAddr:  uint32ToIP(row.LocalAddr),
			localPort:  networkToHostPort(row.LocalPort),
			remoteAddr: uint32ToIP(row.RemoteAddr),
			remotePort: networkToHostPort(row.RemotePort),
			state:      state,
			pid:        int(row.OwningPid),
		})
	}

	return entries, nil
}

// getTCP6Table retrieves TCP IPv6 connections.
func getTCP6Table() ([]connEntry, error) {
	var size uint32
	ret, _, _ := procGetExtendedTcpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET6,
		TCP_TABLE_OWNER_PID_ALL,
		0,
	)
	if ret != 0 && ret != 122 {
		return nil, fmt.Errorf("GetExtendedTcpTable6 size query failed: %d", ret)
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET6,
		TCP_TABLE_OWNER_PID_ALL,
		0,
	)
	if ret != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable6 failed: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	var entries []connEntry

	rowSize := unsafe.Sizeof(tcp6RowOwnerPID{})
	for i := uint32(0); i < numEntries; i++ {
		row := (*tcp6RowOwnerPID)(unsafe.Pointer(&buf[4+uintptr(i)*rowSize]))

		state, ok := winTCPStates[row.State]
		if !ok {
			state = StateUnknown
		}

		localIP := net.IP(row.LocalAddr[:]).String()
		remoteIP := net.IP(row.RemoteAddr[:]).String()

		entries = append(entries, connEntry{
			protocol:   "tcp6",
			localAddr:  localIP,
			localPort:  networkToHostPort(row.LocalPort),
			remoteAddr: remoteIP,
			remotePort: networkToHostPort(row.RemotePort),
			state:      state,
			pid:        int(row.OwningPid),
		})
	}

	return entries, nil
}

// getUDPTable retrieves UDP IPv4 connections.
func getUDPTable() ([]connEntry, error) {
	var size uint32
	ret, _, _ := procGetExtendedUdpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET,
		UDP_TABLE_OWNER_PID,
		0,
	)
	if ret != 0 && ret != 122 {
		return nil, fmt.Errorf("GetExtendedUdpTable size query failed: %d", ret)
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedUdpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET,
		UDP_TABLE_OWNER_PID,
		0,
	)
	if ret != 0 {
		return nil, fmt.Errorf("GetExtendedUdpTable failed: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	var entries []connEntry

	rowSize := unsafe.Sizeof(udpRowOwnerPID{})
	for i := uint32(0); i < numEntries; i++ {
		row := (*udpRowOwnerPID)(unsafe.Pointer(&buf[4+uintptr(i)*rowSize]))

		entries = append(entries, connEntry{
			protocol:   "udp",
			localAddr:  uint32ToIP(row.LocalAddr),
			localPort:  networkToHostPort(row.LocalPort),
			remoteAddr: "0.0.0.0",
			remotePort: 0,
			state:      StateEstablished,
			pid:        int(row.OwningPid),
		})
	}

	return entries, nil
}

// getUDP6Table retrieves UDP IPv6 connections.
func getUDP6Table() ([]connEntry, error) {
	var size uint32
	ret, _, _ := procGetExtendedUdpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET6,
		UDP_TABLE_OWNER_PID,
		0,
	)
	if ret != 0 && ret != 122 {
		return nil, fmt.Errorf("GetExtendedUdpTable6 size query failed: %d", ret)
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedUdpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		AF_INET6,
		UDP_TABLE_OWNER_PID,
		0,
	)
	if ret != 0 {
		return nil, fmt.Errorf("GetExtendedUdpTable6 failed: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	var entries []connEntry

	rowSize := unsafe.Sizeof(udp6RowOwnerPID{})
	for i := uint32(0); i < numEntries; i++ {
		row := (*udp6RowOwnerPID)(unsafe.Pointer(&buf[4+uintptr(i)*rowSize]))

		localIP := net.IP(row.LocalAddr[:]).String()

		entries = append(entries, connEntry{
			protocol:   "udp6",
			localAddr:  localIP,
			localPort:  networkToHostPort(row.LocalPort),
			remoteAddr: "::",
			remotePort: 0,
			state:      StateEstablished,
			pid:        int(row.OwningPid),
		})
	}

	return entries, nil
}

// getProcessName resolves a PID to its executable name on Windows.
func getProcessName(pid int) string {
	if pid == 0 {
		return "System Idle Process"
	}
	if pid == 4 {
		return "System"
	}

	handle, _, err := procOpenProcess.Call(
		PROCESS_QUERY_LIMITED_INFORMATION,
		0,
		uintptr(pid),
	)
	if handle == 0 {
		// Fallback: try reading from /proc-like approach or just return pid-based name
		_ = err
		return fmt.Sprintf("pid:%d", pid)
	}
	defer procCloseHandle.Call(handle)

	// QueryFullProcessImageNameW
	var buf [260]uint16
	size := uint32(len(buf))

	ret, _, _ := procGetProcessImageFileNameW.Call(
		handle,
		0, // use Win32 path format
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return fmt.Sprintf("pid:%d", pid)
	}

	fullPath := syscall.UTF16ToString(buf[:size])
	// Extract just the executable name
	name := filepath.Base(fullPath)
	// Remove .exe suffix for cleaner display
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimSuffix(name, ".EXE")

	return name
}

// isAdmin checks if the current process is running with administrator privileges on Windows.
func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}
