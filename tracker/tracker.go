package tracker

import (
	"strings"
	"sync"
	"time"
)

// Tracker manages the lifecycle of connection tracking.
type Tracker struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	stopCh      chan struct{}
	interval    time.Duration
	pingEnabled bool
}

// NewTracker creates a new Tracker with the given scan interval.
func NewTracker(interval time.Duration, pingEnabled bool) *Tracker {
	return &Tracker{
		connections: make(map[string]*Connection),
		stopCh:      make(chan struct{}),
		interval:    interval,
		pingEnabled: pingEnabled,
	}
}

// Start begins periodic scanning in the background.
func (t *Tracker) Start() {
	// Initial scan
	t.scan()

	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.scan()
			case <-t.stopCh:
				return
			}
		}
	}()
}

// Stop halts the tracker.
func (t *Tracker) Stop() {
	close(t.stopCh)
}

// scan performs a single scan cycle: discover connections, update metrics.
func (t *Tracker) scan() {
	scanned, err := ScanConnections()
	if err != nil {
		return
	}

	now := time.Now()
	t.mu.Lock()

	// Track which keys are still alive
	alive := make(map[string]bool)

	for _, sc := range scanned {
		key := sc.Key()
		alive[key] = true

		existing, ok := t.connections[key]
		if ok {
			// Update existing connection
			existing.State = sc.State
			existing.LastUpdated = now
			existing.ConnAge = now.Sub(existing.FirstSeen)

			// Calculate bandwidth rate
			if !existing.prevTime.IsZero() {
				dt := now.Sub(existing.prevTime).Seconds()
				if dt > 0 {
					if sc.TxBytes >= existing.prevTxBytes {
						existing.TxRate = float64(sc.TxBytes-existing.prevTxBytes) / dt
					}
					if sc.RxBytes >= existing.prevRxBytes {
						existing.RxRate = float64(sc.RxBytes-existing.prevRxBytes) / dt
					}
				}
			}
			existing.prevTxBytes = existing.TxBytes
			existing.prevRxBytes = existing.RxBytes
			existing.prevTime = now
			existing.TxBytes = sc.TxBytes
			existing.RxBytes = sc.RxBytes
		} else {
			// New connection
			sc.FirstSeen = now
			sc.LastUpdated = now
			sc.prevTime = now
			sc.prevTxBytes = sc.TxBytes
			sc.prevRxBytes = sc.RxBytes
			t.connections[key] = sc
		}
	}

	// Remove stale connections
	for key := range t.connections {
		if !alive[key] {
			delete(t.connections, key)
		}
	}

	t.mu.Unlock()

	// Ping in parallel (outside lock)
	if t.pingEnabled {
		t.pingAll()
	}
}

// pingAll measures latency for all active ESTABLISHED connections.
func (t *Tracker) pingAll() {
	t.mu.RLock()
	var targets []*Connection
	for _, c := range t.connections {
		if c.State == StateEstablished && c.RemoteAddr != "0.0.0.0" && c.RemoteAddr != "::" {
			targets = append(targets, c)
		}
	}
	t.mu.RUnlock()

	// Limit concurrency to avoid flooding
	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, c := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(conn *Connection) {
			defer wg.Done()
			defer func() { <-sem }()

			rtt, loss := MeasurePing(conn.RemoteAddr, conn.RemotePort)

			t.mu.Lock()
			conn.Ping = rtt
			conn.PingCount++
			conn.Loss = loss
			if loss >= 100 {
				conn.PingFailed++
			}
			t.mu.Unlock()
		}(c)
	}

	wg.Wait()
}

// Snapshot returns a copy of all current connections.
func (t *Tracker) Snapshot() []*Connection {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*Connection, 0, len(t.connections))
	for _, c := range t.connections {
		cp := *c // shallow copy
		result = append(result, &cp)
	}
	return result
}

// Search returns connections whose AppName contains the given substring (case-insensitive).
func (t *Tracker) Search(query string) []*Connection {
	if query == "" {
		return t.Snapshot()
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	query = strings.ToLower(query)
	var result []*Connection
	for _, c := range t.connections {
		if strings.Contains(strings.ToLower(c.AppName), query) {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result
}
