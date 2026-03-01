# Surveillance Client

## Build & Run
- `make dev` — run Go backend + Vite dev server concurrently
- `make build` — build Go binary with embedded frontend
- `make run` — build and run
- `make download-go2rtc` — download go2rtc binary for current platform

## Project Structure
- Go backend in `main.go` + `internal/`
- React frontend in `web/` (Vite + React 19 + TypeScript + Tailwind v4)
- go2rtc runs as sidecar process managed by the Go backend

## Key Patterns
- Go 1.22+ ServeMux with method+path patterns (no framework)
- SQLite via modernc.org/sqlite (pure Go, no CGO)
- go2rtc reverse-proxied at `/go2rtc/*`
- Frontend uses go2rtc's `<video-rtc>` custom element for MSE streaming
- Zustand for state management

## Testing
- `go test ./...` — run all Go tests
- `cd web && npm test` — run frontend tests

## Environment Variables
- `PORT` — HTTP server port (default: 8080)
- `DB_PATH` — SQLite database path (default: data/surveillance.db)
- `GO2RTC_PATH` — path to go2rtc binary (default: ./go2rtc)
- `GO2RTC_API` — go2rtc API address (default: http://localhost:1984)
- `DVR_HOST` — DVR/camera host for RTSP URL rewriting
- `DVR_USERNAME` — DVR web login username (default: admin)
- `DVR_PASSWORD` — DVR web login password
