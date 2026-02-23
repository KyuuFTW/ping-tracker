# Ping Tracker

A real-time terminal application that monitors all active TCP and UDP connections on your system. It shows which processes are connected to what, measures latency via TCP connect probes, and tracks bandwidth usage -- all in an interactive TUI.

## Usage

Download the latest binary for your platform from the [Releases](../../releases) page. No build step required.

### Linux

```sh
# Download the binary from Releases, then:
chmod +x ping-tracker
sudo ./ping-tracker
```

Root is recommended so the tool can read `/proc/<pid>/fd` to resolve which process owns each connection. It still works without root, but some connections will show as "unknown".

### Windows

Download `ping-tracker.exe` from Releases and run it in a terminal (cmd or PowerShell). Running as Administrator gives full process name resolution.

```
ping-tracker.exe
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-interval` | `3s` | How often connections are rescanned |
| `-no-ping` | `false` | Skip TCP ping probes (faster, no outbound probes) |
| `-filter` | `""` | Pre-filter by app name on startup |

Example:

```sh
sudo ./ping-tracker -interval 5s -filter chrome
```

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` or Arrow keys | Move cursor up/down |
| `g` / `G` | Jump to top / bottom |
| `/` | Start search (filter by app name) |
| `Enter` | Confirm search |
| `Esc` | Cancel search |
| `c` | Clear filter |
| `1`-`6` | Sort by column (press again to reverse) |
| `p` | Pause / resume auto-refresh |
| `r` | Manual refresh |
| `?` | Toggle help screen |
| `q` / `Ctrl+C` | Quit |

## Development

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)

### Building from source

```sh
# Clone the repo
git clone <repo-url>
cd ping-tracker

# Build for your current platform
go build -o ping-tracker .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ping-tracker.exe .
```

### Project structure

```
ping-tracker/
  main.go                      Entry point: CLI flags, bootstrap
  privileges_linux.go           Linux root check
  privileges_windows.go         Windows admin check
  tracker/
    models.go                   Data model: Connection struct, enums, formatters
    tracker.go                  Core engine: scan loop, state reconciliation, ping dispatch
    ping.go                     TCP connect-based latency measurement (cross-platform)
    scanner.go                  Linux scanner: reads /proc/net/tcp{,6} and /proc/net/udp{,6}
    scanner_windows.go          Windows scanner: iphlpapi.dll GetExtendedTcpTable/UdpTable
  tui/
    tui.go                      Terminal UI: Bubble Tea model, rendering, keybindings
```

### Architecture

The app has three layers:

1. **Scanner** (platform-specific) -- Discovers all active connections and resolves owning PIDs. Linux reads the `/proc` filesystem; Windows calls Win32 APIs via `iphlpapi.dll`.

2. **Tracker** (`tracker.go`) -- Runs a background goroutine on a timer. Each cycle it scans connections, reconciles new/updated/stale entries, computes bandwidth rates from byte counter deltas, and dispatches concurrent TCP ping probes.

3. **TUI** (`tui/tui.go`) -- A [Bubble Tea](https://github.com/charmbracelet/bubbletea) application that polls the tracker every 2 seconds for a snapshot, sorts it, and renders a scrollable table with color-coded latency and loss.

### Platform differences

| Feature | Linux | Windows |
|---------|-------|---------|
| Connection scanning | `/proc/net/tcp{,6}`, `/proc/net/udp{,6}` | `GetExtendedTcpTable` / `GetExtendedUdpTable` |
| PID resolution | `/proc/<pid>/fd` inode symlinks | `OpenProcess` + `QueryFullProcessImageNameW` |
| Bandwidth (TX/RX) | Socket queue sizes from `/proc/net` | Not available (always 0 B/s) |
| Ping measurement | TCP connect probe | TCP connect probe |
| Privilege needed | `root` (for full PID resolution) | Administrator (for full process names) |

### Building release binaries

To build the Windows `.exe` for release:

```sh
./build_for_release.sh
```

This produces `ping-tracker.exe` in the project root.

Then create a GitHub release and attach the binaries:

```sh
gh release create v0.1.0 \
  ping-tracker.exe \
  --title "v0.1.0" \
  --notes "Initial release"
```

### Adding a new platform

1. Create `tracker/scanner_<os>.go` with a `//go:build <os>` tag.
2. Implement `func ScanConnections() ([]*Connection, error)`.
3. Create `privileges_<os>.go` with `func checkPrivileges()`.
4. Cross-compile: `GOOS=<os> GOARCH=<arch> go build -o ping-tracker .`
