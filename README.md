# surveillance-client

A modern, self-hosted web client for Viewtron DVR systems. Single binary, real playback speed control, clip export — everything the manufacturer's software should have been.

<!-- TODO: Add a hero GIF here showing the UI in action.
  Record a ~30s screen capture demonstrating:
  1. Live quad view with 4 cameras streaming
  2. Switching to single camera, entering playback mode
  3. Selecting a date/time, playing back at 1x, then jumping to 8x and 32x
  4. Setting In/Out marks and exporting a clip

  Tools: macOS screen recording → gifski (brew install gifski) for high-quality GIF,
  or just use a .mp4 and reference it with GitHub's video embed syntax:
  https://github.com/user-attachments/assets/...
-->

## Why this exists

The Viewtron DVR ships with a web interface that looks like it was built in 2008 — because it was. It uses ActiveX/NPAPI plugins that no modern browser supports, falls back to a sluggish WASM decoder with no speed control, and requires their Windows-only desktop app to export clips. If you want to check your cameras from a Mac or Linux box, you're out of luck.

This project is a from-scratch replacement. It uses [go2rtc](https://github.com/AlexxIT/go2rtc) for low-latency live streaming via MSE (no plugins, no transcoding), reverse-engineers the DVR's proprietary WebSocket protocol for true playback with instant speed changes, and adds clip export that the web UI never had. Everything runs as a single binary with an embedded SPA — deploy it anywhere, open it in any browser.

## Features

- **Live streaming** — Sub-second latency via go2rtc's RTSP→MSE pipeline. No transcoding, no plugins.
- **Quad view** — Watch 4 cameras simultaneously with sub-stream quality, click to go full-resolution.
- **DVR playback** — Date/time picker, 1x–32x speed control with instant switching (ALL_FRAME ≤2x, KEY_FRAME >2x). Skip forward/back by 10s, 1m, or 5m.
- **Clip export** — Set In/Out marks on the timeline, export to MP4. Uses DVR backup mode over WebSocket, WASM decoder for frame assembly, ffmpeg remux on the server.
- **Frame capture** — Grab any frame as JPEG during playback with one click.
- **ONVIF discovery** — Auto-discover cameras on your network using WS-Security PasswordDigest auth. Import channels with correct RTSP URIs.
- **Single binary** — Go binary with embedded React frontend. One file to deploy, nothing to configure beyond env vars.
- **No cloud** — Runs entirely on your LAN. Your video stays on your network.

## Architecture

### Live Streaming

```
┌─────────┐       ┌──────────────────┐       ┌─────────┐       ┌─────────┐
│ Browser  │──────▶│  Go Backend      │──────▶│ go2rtc  │──────▶│  DVR    │
│          │◀──────│  (reverse proxy) │◀──────│ (sidecar)│◀──────│  RTSP   │
│ <video>  │  MSE  │  :8080/go2rtc/*  │  HTTP │  :1984  │ RTSP  │  :554   │
└─────────┘       └──────────────────┘       └─────────┘       └─────────┘
```

The Go backend starts go2rtc as a managed subprocess, generates its config from the camera database, and reverse-proxies its API. The browser uses go2rtc's `<video-rtc>` custom element which negotiates MSE streaming — H.264/H.265 delivered directly from the DVR with no server-side transcoding.

### DVR Playback & Export

```
┌──────────────────────┐         ┌──────────────┐         ┌──────────┐
│ Browser              │         │ Go Backend   │         │ DVR      │
│                      │  WS     │              │  WS     │          │
│ dvr-protocol.ts ─────┼────────▶│ proxy.go ────┼────────▶│ :80/ws   │
│       │              │◀────────┼──────────────┤◀────────┤          │
│       ▼              │         │ (bidirectional         │          │
│ decoder.wasm         │         │  WebSocket proxy)      │          │
│       │              │         │              │         │          │
│       ▼              │         └──────────────┘         └──────────┘
│ webgl-renderer.ts    │
│       │              │
│       ▼              │         ┌──────────────┐
│ <canvas>             │         │ Export Path   │
│                      │         │              │
│ dvr-exporter.ts ─────┼── POST ▶│ ffmpeg       │
│ (AVI assembly)       │  AVI   │ -c copy      │──▶ MP4 download
│                      │◀───────┤ (remux only) │
└──────────────────────┘  MP4   └──────────────┘
```

Playback uses the DVR's proprietary binary WebSocket protocol. The Go backend authenticates with the DVR (`/doLogin`) and proxies WebSocket frames bidirectionally. On the browser side, a WASM decoder (extracted from the manufacturer's web panel) decodes the proprietary frame format, and a WebGL renderer draws to canvas. For export, the browser assembles frames into an AVI container and POSTs it to the backend, where ffmpeg remuxes to MP4 (stream copy, no re-encoding).

## Quick Start

### Prerequisites

- **Go 1.23+** — [go.dev/dl](https://go.dev/dl/)
- **Node.js 20+** — for building the frontend
- **ffmpeg** — required for clip export (`brew install ffmpeg` / `apt install ffmpeg`)
- A Viewtron DVR (tested with VT-DVR-32-4K) accessible on your network

### Build & Run

```bash
# Clone the repo
git clone https://github.com/YOUR_USERNAME/surveillance-client.git
cd surveillance-client

# Download go2rtc for your platform
make download-go2rtc

# Set your DVR connection info
export DVR_HOST=192.168.1.160
export DVR_PASSWORD=your_password

# Build and run (builds frontend + Go binary)
make run
```

Open [http://localhost:8080](http://localhost:8080). Use ONVIF discovery in Settings to find and import your cameras.

### Development

```bash
# Run Go backend + Vite dev server with hot reload
make dev
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `data/surveillance.db` | SQLite database path |
| `GO2RTC_PATH` | `./go2rtc` | Path to go2rtc binary |
| `GO2RTC_API` | `http://localhost:1984` | go2rtc API address |
| `DVR_HOST` | — | DVR IP or hostname (used for RTSP URL rewriting) |
| `DVR_USERNAME` | `admin` | DVR web login username |
| `DVR_PASSWORD` | — | DVR web login password |

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23, stdlib `net/http`, SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) |
| Streaming | [go2rtc](https://github.com/AlexxIT/go2rtc) (RTSP → MSE, managed as sidecar) |
| Frontend | React 19, TypeScript, Vite 6, Tailwind CSS v4, Zustand |
| Playback | Proprietary DVR WebSocket protocol, WASM decoder, WebGL renderer |
| Export | ffmpeg (remux only — stream copy, no re-encoding) |

## How It Works

**Live streaming** — go2rtc connects to the DVR via RTSP on startup, configured from the camera database. The browser uses Media Source Extensions (MSE) for native H.264 playback — no transcoding, no plugins, sub-second latency.

**DVR playback** — The DVR exposes a proprietary binary protocol over WebSocket. The Go backend authenticates via HTTP, then opens a bidirectional WebSocket proxy between the browser and DVR. Commands for play, seek, and speed change are sent as binary frames with channel/timestamp metadata. The WASM decoder (from the DVR's own web panel) decodes the proprietary frame encapsulation, and WebGL renders YUV frames to canvas.

**Clip export** — Uses the DVR's "backup" mode: the browser opens a separate WebSocket session requesting raw frames for a time range, the WASM decoder extracts them, JavaScript assembles an AVI container client-side, and POSTs it to the Go backend where ffmpeg remuxes to MP4 with `-c copy` (no re-encoding, fast).

**ONVIF discovery** — Sends WS-Discovery probes to find devices, then queries ONVIF `GetProfiles` / `GetStreamUri` with WS-Security PasswordDigest authentication. Automatically rewrites internal RTSP URIs (the DVR returns `192.168.x.x` addresses) to the configured `DVR_HOST`.

## License

<!-- TODO: Choose a license. MIT and Apache-2.0 are common for projects like this. -->

<!-- TODO: Screenshots
  Consider adding screenshots of:
  - Settings page with ONVIF discovery results
  - Quad view with 4 live cameras
  - Single camera playback with speed controls and In/Out marks visible
  - Export progress indicator

  Place images in a docs/screenshots/ directory and reference them here.
-->
