# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

DNSMonster is a passive DNS monitoring framework in Go. It captures DNS traffic from live interfaces (afpacket on Linux, libpcap elsewhere), PCAP/PCAPNG files, or dnstap sockets, and dispatches structured DNS events to 15+ output backends at 200k+ queries/second.

## Build & Test

```bash
# Build (requires libpcap-dev on Linux)
go build ./cmd/dnsmonster

# Build without libpcap (no BPF filter support)
go build -tags nolibpcap ./cmd/dnsmonster

# Run all tests
go test ./...

# Single package/test
go test ./internal/capture -run TestPacket

# Tests with race detection (used in CI)
go test -v -race -coverprofile=coverage.out ./...

# Formatting & linting
gofmt -w .
goimports -w .
go vet ./...
```

**Go version**: 1.24+. Build tags: `nolibpcap` disables cgo/BPF. Platform-specific files use `_linux`, `_bsd`, `_windows` suffixes.

## Architecture

Three-stage pipeline, all coordinated via `context.Context` and `errgroup` in `cmd/dnsmonster/main.go`:

1. **Capture** (`internal/capture/`): Reads packets from source, handles IP defragmentation (`defrag.go`), TCP reassembly (`tcpassembly.go`), and DNS parsing (`packet.go`). Produces `DNSResult` structs onto a result channel.

2. **Dispatch** (`cmd/dnsmonster/outputs.go`): Fan-out from the single result channel to all enabled output channels. Per-output filtering (skip/allow domain lists, types 0-4).

3. **Output** (`internal/output/`): Each backend implements the `GenericOutput` interface (defined in `internal/util/types.go`). Consumes from its own channel and writes to the backend.

Shared types and config live in `internal/util/`: `DNSResult` (the core data struct), `GenericOutput` and `OutputMarshaller` interfaces, flag parsing via `go-flags`, domain filtering logic (`functions.go`), and metrics.

## Key Patterns

**Output module registration**: Each output registers itself in `init()` by adding a flag group to `util.GlobalParser` and appending to `util.GlobalDispatchList`. To add a new output, create a file in `internal/output/`, define a config struct with flag tags, implement `GenericOutput`, and register in `init()`.

**Configuration hierarchy** (highest priority first): CLI flags (case-insensitive) > environment variables (`DNSMONSTER_` prefix) > INI config file > defaults. All flags are registered via `github.com/jessevdk/go-flags`.

**Platform-specific capture**: `livecap_linux.go` (afpacket), `livecap_bsd.go` (libpcap), `livecap_windows.go` (npcap). BPF compilation has libpcap/nolibpcap variants via build tags.

## Important Files

- `internal/util/types.go` — `DNSResult`, `GenericOutput`, `OutputMarshaller` interfaces
- `internal/util/util.go` — Flag parsing, logging setup, `GeneralFlags`
- `internal/util/functions.go` — Domain skip/allow filtering logic
- `internal/capture/capture.go` — Capture config and initialization
- `internal/capture/packet.go` — DNS packet parsing from raw bytes
- `cmd/dnsmonster/outputs.go` — Output dispatch and per-output filtering
- `clickhouse/tables.sql` — ClickHouse schema with aggregation views
- `.goreleaser.yaml` — Multi-platform release builds

## Logging

Uses `logrus` with structured fields. Log level controlled by `--loglevel` (0-4, where 4=debug). Supports JSON format via `--logformat=json`.
