package tracker

import (
	"net"
	"testing"
	"time"
)

func TestMeasurePing_SpecialAddresses(t *testing.T) {
	specialAddrs := []string{
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"::",
	}

	for _, addr := range specialAddrs {
		rtt, loss := MeasurePing(addr, 80)
		if rtt != 0 || loss != 0 {
			t.Errorf("expected (0, 0) for special address %q, got rtt=%v, loss=%v", addr, rtt, loss)
		}
	}
}

func TestMeasureTCP_LocalListener(t *testing.T) {
	// Start a local TCP listener to test TCP latency measurement without relying on external network
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create local listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	rtt, loss := measureTCP("127.0.0.1", addr.Port, 2, 500*time.Millisecond)
	if loss != 0 {
		t.Errorf("expected 0%% loss to local TCP listener, got %.1f%%", loss)
	}
	if rtt <= 0 {
		t.Errorf("expected positive RTT to local listener, got %v", rtt)
	}
}

func TestMeasurePing_UnreachableGraceful(t *testing.T) {
	// RFC 5737 TEST-NET-1 (192.0.2.1) is reserved for documentation and will not respond.
	// Verify that MeasurePing completes without panicking and returns loss.
	rtt, loss := MeasurePing("192.0.2.1", 0)
	if loss != 100.0 {
		t.Errorf("expected 100%% loss for unreachable address, got loss=%.1f%%, rtt=%v", loss, rtt)
	}
}
