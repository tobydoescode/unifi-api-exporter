# unifi-api-exporter

Prometheus exporter for UniFi device state. Polls a UniFi OS controller's
`/stat/device` API and exposes each device's numeric `state` — the signal
`unpoller` does not export — so Alertmanager can alert on non-online devices.

## Metrics

| Metric | Labels | Meaning |
|--------|--------|---------|
| `unifi_device_state` | `name, mac, type, model, site` | Controller state (1=connected/online) |
| `unifi_scrape_success` | — | 1 if the last poll succeeded, else 0 |
| `unifi_scrape_duration_seconds` | — | Duration of the last poll |
| `unifi_scrape_errors_total` | — | Total failed polls (counter) |
| `unifi_last_success_timestamp_seconds` | — | Unix time of last successful poll |

On poll failure the `unifi_device_state` gauges retain their last-good values;
use `unifi_last_success_timestamp_seconds` to alert on staleness. Standard
`go_*` and `process_*` runtime metrics are also exported.

### State codes

`1` connected · `0` offline · `2` pending adoption · `4` upgrading ·
`5` provisioning · `6` heartbeat missed · `9` adoption failed · `10` isolated.

## Configuration

| Env | Default | Notes |
|-----|---------|-------|
| `UNIFI_URL` | — | e.g. `https://10.0.0.1` (required) |
| `UNIFI_USER` | — | required |
| `UNIFI_PASS` | — | required |
| `UNIFI_SITE` | `default` | site id |
| `UNIFI_INSECURE` | `false` | set `true` to skip TLS verify (self-signed UDM cert) |
| `POLL_INTERVAL` | `30s` | Go duration; each poll times out after `min(POLL_INTERVAL, 30s)` |
| `LISTEN_ADDR` | `:8080` | metrics/health listener |

Serves `/metrics` and `/healthz`.

After repeated login failures the exporter backs off exponentially (1m doubling
to a 15m cap) to avoid tripping UniFi OS login rate limiting; polls during the
cooldown fail fast and surface via `unifi_scrape_success`.

## Develop

```bash
task test
task lint
task build
```
