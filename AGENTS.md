# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd prime              # Full workflow context
bd hooks install      # Auto-inject bd context
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd create "Title" --type task --priority 2  # Create issue
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Project overview

Dnsmonster is a passive DNS monitoring framework in Go. It captures DNS traffic from live interfaces, PCAP/PCAPNG files, or dnstap sockets, processes packets at high throughput, and emits structured DNS events to configurable outputs while supporting privacy-preserving IP masking.

## Architecture (packet pipeline)

- **Capture** (`internal/capture`): live capture (afpacket on Linux, libpcap elsewhere), pcap/pcapng files, or dnstap sockets. Optional capture-level filters include BPF, sample ratio, and deduplication.
- **Process**: packet parsing, defragmentation/TCP reassembly for DNS over TCP, port filtering, and IPv4/IPv6 masking (`--maskSize4`, `--maskSize6`).
- **Dispatch** (`internal/output/output.go`): each processed event is fan-out to every configured output module.
- **Output filters**: per-output Type 0–4 controls allow/skip domain filters; output formats include json, csv, csv_no_header, gotemplate, and json-ocsf where applicable.
- **Metrics** (`internal/util/metrics.go`): go-metrics with stderr, statsd, or prometheus backends.

## Outputs and storage

Primary storage is ClickHouse (schema in `clickhouse/tables.sql`) with Grafana dashboards in `grafana/panel.json`. Other outputs live in `internal/output` and include stdout/file/syslog, Kafka, PostgreSQL, Elastic, Influx, Parquet, Splunk, Sentinel, VictoriaLogs, and Zinc.

## Repository map

- `cmd/dnsmonster/`: main entrypoint.
- `internal/capture/`: capture, parsing, and packet pipeline.
- `internal/output/`: output modules and dispatcher.
- `internal/util/`: formatting, metrics, and shared helpers.
- `clickhouse/`: schema, dictionaries, and docker-compose for ClickHouse+Grafana.
- `docs/`: Hugo docs site (`hugo serve` from `docs/`).

## Build and test

- Build (with libpcap): `go build -o dnsmonster ./cmd/dnsmonster`
- Build (no libpcap/BPF): `go build -tags nolibpcap -o dnsmonster ./cmd/dnsmonster`
- Tests: `go test ./...`

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
