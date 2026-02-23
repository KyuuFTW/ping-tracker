//go:build linux

package tracker

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// procStates maps the hex state codes in /proc/net/tcp to ConnState.
var procStates = map[string]ConnState{
	"01": StateEstablished,
	"02": StateSynSent,
	"03": StateSynRecv,
	"04": StateFinWait1,
	"05": StateFinWait2,
	"06": StateTimeWait,
	"07": StateClosed,
	"08": StateCloseWait,
	"09": StateLastAck,
	"0A": StateListening,
	"0B": StateClosing,
}

// inodeEntry holds a parsed /proc/net line before PID resolution.
type inodeEntry struct {
	protocol   string
	localAddr  string
	localPort  int
	remoteAddr string
	remotePort int
	state      ConnState
	inode      string
	txQueue    uint64
	rxQueue    uint64
}

// ScanConnections reads /proc/net/tcp and /proc/net/tcp6 to discover connections,
// then resolves each socket inode to a PID and process name.
func ScanConnections() ([]*Connection, error) {
	now := time.Now()

	var entries []inodeEntry

	for _, proto := range []string{"tcp", "tcp6"} {
		path := "/proc/net/" + proto
		parsed, err := parseProcNet(path, proto)
		if err != nil {
			continue // skip if file doesn't exist (e.g., no IPv6)
		}
		entries = append(entries, parsed...)
	}

	// Build inode -> PID+name map
	inodePID, inodeName := buildInodeMap()

	// Also read UDP for completeness
	for _, proto := range []string{"udp", "udp6"} {
		path := "/proc/net/" + proto
		parsed, err := parseProcNet(path, proto)
		if err != nil {
			continue
		}
		entries = append(entries, parsed...)
	}

	var conns []*Connection
	for _, e := range entries {
		pid := inodePID[e.inode]
		name := inodeName[e.inode]
		if name == "" {
			name = "unknown"
		}

		dir := Outbound
		if e.state == StateListening || e.remoteAddr == "0.0.0.0" || e.remoteAddr == "::" {
			dir = Inbound
		}

		conn := &Connection{
			PID:         pid,
			AppName:     name,
			Protocol:    e.protocol,
			Direction:   dir,
			LocalAddr:   e.localAddr,
			LocalPort:   e.localPort,
			RemoteAddr:  e.remoteAddr,
			RemotePort:  e.remotePort,
			State:       e.state,
			TxBytes:     e.txQueue,
			RxBytes:     e.rxQueue,
			FirstSeen:   now,
			LastUpdated: now,
		}
		conns = append(conns, conn)
	}

	return conns, nil
}

// parseProcNet parses a /proc/net/tcp or /proc/net/udp file.
func parseProcNet(path, protocol string) ([]inodeEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []inodeEntry
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // skip header
		}
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		localAddr, localPort, err := parseAddr(fields[1])
		if err != nil {
			continue
		}
		remoteAddr, remotePort, err := parseAddr(fields[2])
		if err != nil {
			continue
		}

		stateHex := strings.ToUpper(fields[3])
		state, ok := procStates[stateHex]
		if !ok {
			state = StateUnknown
		}

		// tx_queue:rx_queue
		queues := strings.Split(fields[4], ":")
		var txQ, rxQ uint64
		if len(queues) == 2 {
			txQ, _ = strconv.ParseUint(queues[0], 16, 64)
			rxQ, _ = strconv.ParseUint(queues[1], 16, 64)
		}

		inode := fields[9]

		entries = append(entries, inodeEntry{
			protocol:   protocol,
			localAddr:  localAddr,
			localPort:  localPort,
			remoteAddr: remoteAddr,
			remotePort: remotePort,
			state:      state,
			inode:      inode,
			txQueue:    txQ,
			rxQueue:    rxQ,
		})
	}

	return entries, nil
}

// parseAddr parses a hex-encoded address:port like "0100007F:0035" from /proc/net.
func parseAddr(s string) (string, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid addr: %s", s)
	}

	port, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return "", 0, err
	}

	addrHex := parts[0]
	addr, err := hexToIP(addrHex)
	if err != nil {
		return "", 0, err
	}

	return addr, int(port), nil
}

// hexToIP converts a hex-encoded IP from /proc/net to a string.
func hexToIP(h string) (string, error) {
	b, err := hex.DecodeString(h)
	if err != nil {
		return "", err
	}

	switch len(b) {
	case 4:
		// IPv4 - /proc stores in little-endian 32-bit word
		ip := net.IPv4(b[3], b[2], b[1], b[0])
		return ip.String(), nil
	case 16:
		// IPv6 - stored as 4 little-endian 32-bit words
		ip := make(net.IP, 16)
		for i := 0; i < 4; i++ {
			word := b[i*4 : i*4+4]
			binary.BigEndian.PutUint32(ip[i*4:i*4+4],
				binary.LittleEndian.Uint32(word))
		}
		return ip.String(), nil
	default:
		return "", fmt.Errorf("unexpected addr length: %d", len(b))
	}
}

// buildInodeMap scans /proc/*/fd/* to map socket inodes to PIDs and process names.
func buildInodeMap() (map[string]int, map[string]string) {
	inodePID := make(map[string]int)
	inodeName := make(map[string]string)

	procs, _ := filepath.Glob("/proc/[0-9]*/fd/[0-9]*")
	for _, fdPath := range procs {
		link, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(link, "socket:[") {
			continue
		}
		inode := link[8 : len(link)-1]

		// Extract PID from path: /proc/<pid>/fd/<fd>
		parts := strings.Split(fdPath, "/")
		if len(parts) < 4 {
			continue
		}
		pid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		inodePID[inode] = pid

		// Read process name from /proc/<pid>/comm
		commPath := fmt.Sprintf("/proc/%d/comm", pid)
		comm, err := os.ReadFile(commPath)
		if err == nil {
			inodeName[inode] = strings.TrimSpace(string(comm))
		}
	}

	return inodePID, inodeName
}
