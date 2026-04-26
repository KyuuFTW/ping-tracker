package main

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"ping-tracker/tracker"
)

// App is the Wails backend bridge. It owns the tracker lifecycle and exposes
// JSON-safe DTO methods to the web frontend.
type App struct {
	ctx context.Context

	mu            sync.RWMutex
	tracker       *tracker.Tracker
	interval      time.Duration
	pingEnabled   bool
	initialFilter string
	paused        bool
}

func NewApp(interval time.Duration, pingEnabled bool, initialFilter string) *App {
	return &App{
		tracker:       tracker.NewTracker(interval, pingEnabled),
		interval:      interval,
		pingEnabled:   pingEnabled,
		initialFilter: initialFilter,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	checkPrivileges()
	a.tracker.Start()
}

func (a *App) shutdown(ctx context.Context) {
	a.tracker.Stop()
}

func (a *App) GetInitialState() InitialStateDTO {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ready, scanning, lastScan, lastError := a.tracker.Status()

	return InitialStateDTO{
		Filter:      a.initialFilter,
		IntervalMS:  int(a.interval / time.Millisecond),
		PingEnabled: a.pingEnabled,
		Paused:      a.paused,
		Ready:       ready,
		Scanning:    scanning,
		LastScan:    lastScan.UnixMilli(),
		LastError:   lastError,
	}
}

func (a *App) GetStatus() StatusDTO {
	ready, scanning, lastScan, lastError := a.tracker.Status()
	return StatusDTO{
		Ready:     ready,
		Scanning:  scanning,
		LastScan:  lastScan.UnixMilli(),
		LastError: lastError,
	}
}

func (a *App) GetConnections(filter string, sortField string, sortAsc bool) []ConnectionDTO {
	a.mu.RLock()
	paused := a.paused
	a.mu.RUnlock()

	var conns []*tracker.Connection
	if filter != "" {
		conns = a.tracker.Search(filter)
	} else {
		conns = a.tracker.Snapshot()
	}

	sortConnections(conns, sortField, sortAsc)

	result := make([]ConnectionDTO, 0, len(conns))
	for _, c := range conns {
		result = append(result, connectionDTO(c, paused))
	}
	return result
}

func (a *App) GetHistory(key string) []PingSampleDTO {
	if key == "" {
		return nil
	}

	for _, c := range a.tracker.Snapshot() {
		if c.Key() != key {
			continue
		}

		result := make([]PingSampleDTO, 0, len(c.PingHistory))
		for _, s := range c.PingHistory {
			result = append(result, PingSampleDTO{
				Time:  s.Time.UnixMilli(),
				RTTMS: durationMS(s.RTT),
				Loss:  s.Loss,
			})
		}
		return result
	}

	return nil
}

func (a *App) Pause() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.paused = true
	return a.paused
}

func (a *App) Resume() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.paused = false
	return a.paused
}

func (a *App) IsPaused() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.paused
}

func (a *App) Refresh() []ConnectionDTO {
	return a.GetConnections("", "app", true)
}

func sortConnections(conns []*tracker.Connection, field string, asc bool) {
	sort.SliceStable(conns, func(i, j int) bool {
		a, b := conns[i], conns[j]
		cmp := 0

		switch field {
		case "ping":
			cmp = compareDuration(a.Ping, b.Ping)
		case "loss":
			cmp = compareFloat(a.Loss, b.Loss)
		case "tx":
			cmp = compareFloat(a.TxRate, b.TxRate)
		case "rx":
			cmp = compareFloat(a.RxRate, b.RxRate)
		case "state":
			cmp = strings.Compare(string(a.State), string(b.State))
		case "pid":
			cmp = compareInt(a.PID, b.PID)
		default:
			cmp = strings.Compare(strings.ToLower(a.AppName), strings.ToLower(b.AppName))
		}

		if !asc {
			cmp = -cmp
		}
		if cmp != 0 {
			return cmp < 0
		}
		if a.Direction != b.Direction {
			return a.Direction == tracker.Outbound
		}
		return a.Key() < b.Key()
	})
}

func compareDuration(a, b time.Duration) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
