# Plan: Go Performance Audit — Dual-Model Expert Review

## Problem

The waza codebase (239 Go files, ~53K LOC) has never had a dedicated performance audit. Key areas — orchestration engine, Copilot SDK execution, JSON-RPC transport, caching, tokenizer, and Azure service interactions — may have opportunities for latency reduction, better concurrency, reduced allocations, and faster I/O.

## Approach

Two-phase expert review with model diversity for maximum coverage:

1. **Phase 1 — GPT-5.3-Codex deep audit** (Go performance specialist)
2. **Phase 2 — Claude Opus 4.6 follow-up review** (independent second opinion)
3. **Phase 3 — Synthesize and compare** (coordinator merges findings)

## New Team Member

Add **Turk** — Go Performance Specialist — to the squad roster (Ocean's Eleven universe, already in use). Turk's charter focuses on: profiling, allocation analysis, concurrency patterns, I/O optimization, and Azure SDK interaction efficiency.

Turk is a permanent addition. Go performance is an ongoing concern as waza grows.

## Audit Scope

### Hot-Path Files (highest impact)

| File | Lines | Why |
|------|-------|-----|
| `internal/orchestration/runner.go` | 1548 | Core test runner — goroutines, file I/O, fixture copying |
| `cmd/waza/cmd_run.go` | 1296 | CLI entry point for evals — flag parsing, result collection |
| `internal/execution/copilot.go` | ~400 | Copilot SDK integration — HTTP, JSON-RPC, session mgmt |
| `internal/jsonrpc/transport.go` | ~200 | JSON-RPC transport — serialization, connection mgmt |
| `internal/jsonrpc/handlers.go` | ~300 | Request/response handling — potential allocation hotspot |
| `internal/cache/cache.go` | ~300 | Caching layer — concurrency, eviction, serialization |
| `internal/tokens/bpe/tokenizer.go` | 616 | BPE tokenizer — CPU-bound, string processing |
| `internal/webserver/server.go` | ~200 | HTTP server — middleware, routing |
| `internal/webapi/handlers.go` | ~300 | API handlers — JSON marshaling, store interactions |
| `internal/webapi/store.go` | ~200 | Data store — concurrent access patterns |

### Azure-Specific Files

| File | Concern |
|------|---------|
| `internal/execution/copilot.go` | Azure-hosted Copilot SDK — connection pooling, retry, timeout |
| `internal/execution/session_events_collector.go` | Event stream processing — buffering, backpressure |
| `cmd/waza/cmd_metadata.go` | Azure metadata queries |
| `internal/checks/token_limits.go` | Token budget checking against Azure limits |

### Cross-Cutting Concerns

- **Concurrency:** 21+ files use goroutines, mutexes, channels, WaitGroups
- **JSON serialization:** 83+ files with encoding/json — potential for faster alternatives
- **File I/O:** fixture copying in orchestration, config walking, file-based graders
- **HTTP clients:** link checking, suggestion APIs, web server — connection reuse, timeouts

## Todos

### Phase 1: GPT-5.3-Codex Performance Audit
- `add-turk` — Add Turk (Go Performance Specialist) to the squad roster
- `codex-audit` — Spawn Turk with GPT-5.3-Codex to audit all hot-path files, concurrency patterns, allocation hotspots, Azure interactions, and I/O bottlenecks

### Phase 2: Claude Opus 4.6 Follow-Up Review
- `opus-review` — Spawn Turk with Claude Opus 4.6 to independently review the same scope

### Phase 3: Synthesis
- `compare-findings` — Compare both reports, produce unified recommendations with agreement/disagreement markers, prioritized by impact

## Output

A single markdown document with:
- Findings from GPT-5.3-Codex (labeled)
- Findings from Claude Opus 4.6 (labeled)
- Comparison table: which findings overlap, which are unique to each model
- Prioritized action list (P0/P1/P2) with specific file references
