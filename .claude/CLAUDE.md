# Ping Tracker - Maintainer Context

## What this project is

A real-time terminal network monitor written in Go. It discovers all TCP/UDP connections on the system, resolves which process owns each one, measures latency via TCP connect probes, and displays everything in an interactive TUI.

## Key design decisions

### Platform abstraction via build tags

The scanner and privilege check are the only platform-specific code. They are isolated using Go build tags (`//go:build linux` / `//go:build windows`). Everything else (tracker engine, ping measurement, TUI) is cross-platform.

- `tracker/scanner.go` -- Linux only. Parses `/proc/net/tcp{,6}` and `/proc/net/udp{,6}`. Resolves PIDs by scanning `/proc/*/fd/*` symlinks for socket inodes.
- `tracker/scanner_windows.go` -- Windows only. Calls `GetExtendedTcpTable`/`GetExtendedUdpTable` from `iphlpapi.dll`. Resolves process names via `OpenProcess` + `QueryFullProcessImageNameW`.
- `privileges_linux.go` / `privileges_windows.go` -- Platform-specific privilege warnings.

Both scanners implement the same function: `func ScanConnections() ([]*Connection, error)`.

### TCP ping instead of ICMP

Ping measurement uses `net.DialTimeout("tcp", ...)` rather than ICMP. This avoids requiring raw socket privileges and works on both platforms without any special permissions. The tradeoff is it only works against ports that accept TCP connections.

### ANSI-safe column alignment

The TUI pads plain text to column width *before* applying lipgloss styles. This is critical -- `fmt.Sprintf("%-*s", width, styledText)` does not work because ANSI escape codes are counted as visible characters by Go's formatter. See `padRight()` and `styledPadRight()` in `tui/tui.go`.

### Bandwidth tracking is Linux-only

The Linux scanner reads `tx_queue:rx_queue` from `/proc/net/tcp` to get byte counters. The Windows `GetExtendedTcpTable` API does not expose byte counters, so TX/RX rates always show `0 B/s` on Windows. This is a known limitation, not a bug.

## Concurrency model

- `Tracker` runs a background goroutine on a `time.Ticker` (default 3s).
- Each scan cycle holds a write lock (`sync.RWMutex`) during connection reconciliation.
- Ping probes run outside the lock, each goroutine briefly acquires a write lock to update results. Concurrency is capped at 20 goroutines via a semaphore channel.
- The TUI calls `Snapshot()`/`Search()` under a read lock, which returns shallow copies so rendering never blocks the scanner.

## Column layout

Columns are ordered with the most useful info first: PID, App, Ping, Loss, then Dir, Proto, endpoints, State, TX, RX. Sort keys `1`-`6` map to: App, Ping, Loss, TX, RX, State. The secondary sort always places Outbound (`OUT`) above Inbound (`IN`) when the primary sort field is tied.

## How to add a new platform

1. Create `tracker/scanner_<os>.go` with `//go:build <os>` and implement `ScanConnections()`.
2. Create `privileges_<os>.go` with `//go:build <os>` and implement `checkPrivileges()`.
3. Cross-compile: `GOOS=<os> GOARCH=<arch> go build`.

## Build and test

```sh
# Native build
go build -o ping-tracker .

# Windows cross-compile (no CGO needed)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ping-tracker.exe .
```

There are no tests currently. The app is verified manually by running it and inspecting the TUI output.

## Dependencies

All external deps are from the Charm ecosystem (`bubbletea` for the TUI framework, `lipgloss` for styling) plus `golang.org/x/sys` for low-level OS primitives. No other third-party libraries.
