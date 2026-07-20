# AGENTS.md

Go Prometheus exporter for UniFi device state.

- `internal/config` — env config loading.
- `internal/unifi` — UniFi OS controller client (login + `/stat/device`).
- `internal/collector` — Prometheus metrics.
- `main.go` — HTTP server + poll loop.

## Conventions

- TDD: write the test first, keep packages small and focused.
- Read-only against the controller. Never issue mutating API calls.
- Keep the image distroless/nonroot; no CGO.
- Run `task test` and `task lint` before committing.
