# Ping Tracker

A lightweight Windows-focused desktop app that monitors active TCP and UDP connections, measures TCP-connect latency, tracks packet loss, and shows selected connection history as graphs.

The app now uses [Wails](https://wails.io/) for a clickable desktop UI with the existing Go tracker running in the background.

## Usage

Download the latest binary for your platform from the [Releases](../../releases) page.

### Windows

Download `ping-tracker.exe` and run it normally. Running as Administrator gives full process name resolution.

```powershell
.\ping-tracker.exe
```

Wails uses Microsoft Edge WebView2 on Windows. Most Windows 10/11 systems already include it; otherwise install the WebView2 runtime from Microsoft.

### Linux

```sh
chmod +x ping-tracker
sudo ./ping-tracker
```

Root is recommended so the tool can read `/proc/<pid>/fd` to resolve which process owns each connection. It still works without root, but some connections will show as `unknown`.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-interval` | `3s` | How often connections are rescanned |
| `-no-ping` | `false` | Skip TCP ping probes |
| `-filter` | `""` | Initial app name filter |

Example:

```sh
./ping-tracker -interval 5s -filter chrome
```

## Features

| Feature | Description |
|---------|-------------|
| Connection table | App, PID, ping, loss, direction, protocol, local/remote endpoints, state, TX/RX |
| Click selection | Select a row to show detail and history |
| Search | Filter connections by app name |
| Sorting | Click table headers to sort |
| Pause | Pause UI refresh without stopping the app |
| Graphs | Ping latency and packet loss over the last 5 minutes |

## Development

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 22+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

Linux development also needs WebKitGTK packages for local `wails dev` and Linux builds:

```sh
sudo apt install pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev
```

Windows release builds from Linux do not require those Linux GUI packages.

Install Wails CLI:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Install frontend dependencies:

```sh
cd frontend
npm install
```

### Development Server

Run the desktop app with hot reload:

```sh
wails dev
```

Run only the web frontend during UI iteration:

```sh
cd frontend
npm run dev
```

The standalone Vite server is useful for layout work, but backend data is only available inside Wails.

### Verification

Before opening a pull request or making a release, run:

```sh
cd frontend
npm run build
cd ..
go build ./...
wails build -platform windows/amd64 -clean
```

The native Linux Wails build needs the WebKitGTK development packages listed above.

### Build

Build for the current platform:

```sh
wails build
```

Build for Windows:

```sh
wails build -platform windows/amd64 -clean
```

The Wails Windows binary is written to:

```text
build/bin/ping-tracker.exe
```

### Release Artifact

Use the release helper to produce a versioned `.exe` that can be uploaded directly to GitHub Releases:

```sh
./build_for_release.sh v0.1.0
```

This creates:

```text
dist/release/ping-tracker-windows-amd64-v0.1.0.exe
dist/release/ping-tracker-windows-amd64-v0.1.0.exe.sha256
```

If no version is passed, the artifact uses `snapshot`:

```sh
./build_for_release.sh
```

To create a GitHub release and upload the Windows executable:

```sh
gh release create v0.1.0 \
  dist/release/ping-tracker-windows-amd64-v0.1.0.exe \
  dist/release/ping-tracker-windows-amd64-v0.1.0.exe.sha256 \
  --title "v0.1.0" \
  --notes "Windows desktop release. Requires Microsoft Edge WebView2 Runtime."
```

Users can download the `.exe` asset from the release page and run it directly. Running as Administrator is recommended for full process name resolution.

## Project Structure

```text
ping-tracker/
  main.go                    Wails bootstrap and CLI flags
  app.go                     Wails backend methods
  dto.go                     JSON-safe frontend DTOs
  privileges_linux.go        Linux root warning
  privileges_windows.go      Windows admin warning
  tracker/
    models.go                Data model and formatters
    tracker.go               Scan loop, reconciliation, ping dispatch
    ping.go                  TCP connect latency measurement
    scanner.go               Linux scanner
    scanner_windows.go       Windows scanner
  frontend/
    src/App.svelte           Main UI
    src/components/Graph.svelte
    src/lib/api.ts           Backend bridge wrapper
    src/lib/types.ts         Frontend DTO types
```

## Architecture

1. **Scanner** discovers active connections. Linux reads `/proc`; Windows calls `iphlpapi.dll` APIs.
2. **Tracker** runs a background timer, reconciles connections, computes rates, and records ping/loss history.
3. **Wails Backend** exposes safe snapshot/history methods to the frontend.
4. **Frontend** polls the backend every 1.5 seconds and renders the clickable table and graphs.

## Platform Differences

| Feature | Linux | Windows |
|---------|-------|---------|
| Connection scanning | `/proc/net/tcp{,6}`, `/proc/net/udp{,6}` | `GetExtendedTcpTable` / `GetExtendedUdpTable` |
| PID resolution | `/proc/<pid>/fd` inode symlinks | `OpenProcess` + `QueryFullProcessImageNameW` |
| Bandwidth | Socket queue sizes from `/proc/net` | Not available, currently `0 B/s` |
| Ping measurement | TCP connect probe | TCP connect probe |
| Privilege needed | root for full PID resolution | Administrator for full process names |
