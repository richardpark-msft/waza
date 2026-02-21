# waza × MSBench Integration Design

**Date:** February 21, 2026
**Author:** Rusty (Lead/Architect), requested by Shayne Boyer + Peter
**Status:** Draft for MSBench team conversation
**Context:** waza v0.8.0 · MSBench (devdiv-microsoft/MicrosoftSweBench)

---

## Thesis

**Inner loop / outer loop.** What you run locally with waza should be flavors of what you run at scale in MSBench. waza is the authoring + local dev tool. MSBench provides distributed compute to run the same evals 100× at scale.

This isn't two competing platforms — it's one evaluation pipeline with two execution modes.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        EVAL DEFINITION LAYER                            │
│                                                                         │
│    eval.yaml ──► tasks/*.yaml ──► graders ──► fixtures/                 │
│         ▲              ▲              ▲                                  │
│         │              │              │                                  │
│    waza init      waza new       waza suggest                           │
│    waza dev       waza check     waza tokens                            │
│                                                                         │
└───────────────────────┬─────────────────────────────────────────────────┘
                        │
                   eval bundle
                   (portable)
                        │
          ┌─────────────┼─────────────┐
          │             │             │
          ▼             │             ▼
┌─────────────────┐     │     ┌─────────────────────────────────────────┐
│  INNER LOOP     │     │     │  OUTER LOOP                             │
│  (waza local)   │     │     │  (MSBench cloud)                        │
│                 │     │     │                                         │
│  1-5 trials     │     │     │  50-100 runs                            │
│  seconds/min    │     │     │  Harbor containers                      │
│  laptop/CI      │     │     │  Azure compute fleet                    │
│  JSON results   │     │     │  Kusto data layer                       │
│  dashboard UI   │     │     │  production-like envs                   │
│                 │     │     │                                         │
│  waza run       │     │     │  msbench-cli run                        │
│  waza serve     │     │     │  msbench-cli report                     │
│  waza compare   │     │     │  msbenchapp.azurewebsites.net           │
└────────┬────────┘     │     └──────────────┬──────────────────────────┘
         │              │                    │
         │    ┌─────────┴──────────┐         │
         │    │  BRIDGE LAYER      │         │
         │    │                    │         │
         └───►│  waza export       │◄────────┘
              │  waza import       │
              │  format adapters   │
              │  grader shims      │
              └────────────────────┘
                        │
                        ▼
              ┌────────────────────┐
              │  UNIFIED RESULTS   │
              │                    │
              │  waza dashboard    │
              │  trajectory view   │
              │  compare view      │
              │  trends over time  │
              └────────────────────┘
```

---

## 1. The Two-Loop Model

### Inner Loop: waza (local + CI)

**When:** Every commit, every skill edit, every prompt tweak. The skill author's daily driver.

| Dimension | Value |
|-----------|-------|
| Trials | 1-5 per task |
| Latency | Seconds to low minutes |
| Compute | Laptop or CI runner |
| Output | `results.json` → dashboard |
| Feedback | Immediate via `waza serve` |
| Executor | `mock` (instant) or `copilot-sdk` (real) |

**Workflow:** `waza init` → edit skill → `waza run` → `waza serve` → inspect trajectory → `waza suggest` → iterate → `waza check` → commit.

### Outer Loop: MSBench (cloud)

**When:** Before shipping a skill. When you need statistical confidence. When you need production-like environments with real repos, real file systems, real tool chains.

| Dimension | Value |
|-----------|-------|
| Runs | 50-100 per benchmark instance |
| Latency | Minutes to hours |
| Compute | Azure fleet, Harbor containers |
| Output | Kusto tables → MSBench web UI |
| Environments | Containerized, production-like |
| Executor | CES CAPI proxy, special agents shim |

**Workflow:** `waza export --target msbench` → push to MSBench → run at scale → `waza import --from msbench` → analyze in waza dashboard.

### Flow from Inner → Outer

The eval definition is the shared contract. A skill author writes `eval.yaml` + `tasks/*.yaml` + graders in waza. When ready for scale:

```
# Author locally
waza run eval.yaml -v                     # inner loop: 3 trials, fast feedback

# Export for scale
waza export --target msbench -o bench/    # generates Harbor-compatible config

# Run at scale (MSBench side)
msbench-cli run --config bench/           # 100 runs, containerized envs

# Pull results back
waza import --from msbench --run-id abc   # Kusto → waza JSON
waza serve                                # unified dashboard view
```

**Key principle:** The eval definition stays in waza format. The bridge translates at export/import time. Skill authors never write MSBench configs directly.

---

## 2. Eval Format Bridge

### The Problem

waza and MSBench use fundamentally different eval specs:

| Dimension | waza | MSBench |
|-----------|------|---------|
| Spec format | `eval.yaml` + `tasks/*.yaml` | Harbor benchmark config (TOML/YAML) |
| Task definition | YAML with prompt, expected, context_files | Containerized task instance with repo + golden patch |
| Fixtures | Directory of files, copied per task | Full container image with toolchain |
| Grading | 11 Go validator types | Grader SDK v0 (custom scripts, env-agnostic) |
| Execution | copilot-sdk or mock | CES CAPI proxy + special agents shim |

### The Translation Layer

`waza export --target msbench` produces a Harbor-compatible benchmark package:

```
bench/
├── benchmark.yaml          # MSBench benchmark manifest
├── instances/
│   ├── task-001/
│   │   ├── instance.yaml   # MSBench instance config
│   │   ├── prompt.md       # Extracted from waza task
│   │   ├── fixtures/       # Copied from --context-dir
│   │   └── graders/        # Wrapped waza graders as scripts
│   └── task-002/
│       └── ...
├── Dockerfile              # Base container with waza grader runtime
└── grader-shim.sh          # Bridges waza validators → MSBench grader protocol
```

**Mapping rules:**

| waza concept | MSBench equivalent | Translation |
|---|---|---|
| `eval.yaml` | `benchmark.yaml` | Direct field mapping (name, description, config) |
| `tasks/*.yaml` | `instances/*/instance.yaml` | One task → one instance. Prompt extracted, fixtures copied |
| `config.model` | Agent config / model selection | Passed through to MSBench agent config |
| `config.timeout_seconds` | Instance timeout | Direct mapping |
| `config.trials_per_task` | `repeat_count` on instance | Direct mapping |
| `graders[]` | `grader-shim.sh` per instance | Waza graders wrapped as executable scripts (see §4) |
| `context_files` | Instance fixture directory | Files copied into container |
| `expected.contains` | Grader assertion (via shim) | Translated to grader script check |

### What `waza export` Does NOT Do

- Does not build container images (MSBench team owns their Harbor pipeline)
- Does not push to MSBench storage (that's `msbench-cli run`)
- Does not translate waza executors → MSBench agents (execution environments differ)

### The Minimal Export: eval.yaml → benchmark.yaml

For the first iteration, the export could be as simple as:

```yaml
# Generated benchmark.yaml (MSBench format)
name: code-explainer-eval
source: waza
waza_version: 0.8.0
repeat_count: 100
timeout_seconds: 300

instances:
  - id: explain-python-recursion
    prompt_file: instances/explain-python-recursion/prompt.md
    fixtures_dir: instances/explain-python-recursion/fixtures/
    grader: instances/explain-python-recursion/grader-shim.sh
    metadata:
      waza_task: explain-python-recursion
      waza_graders: ["keyword", "code"]
```

**Effort estimate:** 1 dev / 2-3 weeks for the core export command. Another 1-2 weeks for grader shim packaging.

---

## 3. Results Flow

### The Problem

MSBench stores results in Kusto. waza dashboard expects JSON. Different schemas, different query models.

### Kusto → waza JSON

`waza import --from msbench --run-id <id>` does:

1. **Query Kusto** for the run's results (using MSBench's KQL functions)
2. **Map fields** to waza's result schema
3. **Write** `results.json` in waza format
4. **Merge** with local runs (tagged as `source: msbench`)

```
MSBench Kusto Schema              waza results.json
─────────────────────             ──────────────────
RunId                    ───►     eval_id
BenchmarkName            ───►     eval_name
InstanceId               ───►     tasks[].id
ResolveStatus            ───►     tasks[].status (passed/failed)
TokenCount               ───►     tasks[].trials[].token_count
Duration                 ───►     tasks[].trials[].duration_ms
GraderOutput             ───►     tasks[].trials[].grader_results
Trajectory (if ATIF)     ───►     tasks[].trials[].trajectory
ModelName                ───►     config.model
Timestamp                ───►     timestamp
```

### What This Enables

- **100-run statistical view in waza dashboard**: Bootstrap CI, significance badges, pass rate distributions — computed by waza's pipeline from the 100 imported trials
- **Trajectory comparison**: If MSBench captures ATIF trajectories, waza's waterfall timeline can render them (format bridge needed for ATIF → waza trajectory format)
- **Mixed-source trends**: The TrendsPage shows local runs and MSBench runs on the same timeline, distinguishable by `source` tag

### Authentication

MSBench's Kusto requires Entra ID. Options:
- **Option A:** `waza import` shells out to `msbench-cli extract` (reuses their auth)
- **Option B:** `waza import` takes a Kusto connection string + `az login` token
- **Recommendation:** Option A. Don't reinvent their auth. `msbench-cli extract --run-id <id> --format json > results.json` might already be close enough — we just need the format adapter.

### Effort Estimate

- Kusto schema mapping: 1 dev / 1 week
- `waza import` command: 1 dev / 1 week
- Trajectory format bridge (ATIF → waza): 1 dev / 1-2 weeks (depends on ATIF stability)
- Total: 1 dev / 3-4 weeks

---

## 4. Grader Compatibility

### The Problem

waza has 11 grader types implemented as Go `Validator` interface implementations. MSBench is designing a Grader SDK v0 that aims for environment-agnostic custom graders. Neither side's graders run natively in the other's environment.

### waza Graders Today

```go
type Validator interface {
    Validate(ctx context.Context, input *ValidationInput) (*ValidationResult, error)
}
```

11 types: `keyword`, `regex`, `code`, `file`, `diff`, `json_schema`, `program`, `inline_script`, `prompt`, `behavior`, `action_sequence`, `skill_invocation`.

### Compatibility Strategy: Grader Shim

Package waza graders as executable scripts that MSBench can invoke through its Grader SDK:

```
┌──────────────────────────────────────────────────────┐
│  MSBench Container                                    │
│                                                       │
│  ┌────────────────┐     ┌──────────────────────────┐ │
│  │ MSBench Grader  │────►│ grader-shim (waza binary)│ │
│  │ SDK v0 hook     │     │                          │ │
│  │                 │◄────│ Reads: output, fixtures   │ │
│  │ Pass/Fail +     │     │ Runs: waza validators     │ │
│  │ score + details │     │ Returns: JSON verdict     │ │
│  └────────────────┘     └──────────────────────────┘ │
│                                                       │
└──────────────────────────────────────────────────────┘
```

**The shim is a stripped-down waza binary** that:
1. Reads the agent's output from stdin or file
2. Reads grader config from a sidecar YAML
3. Runs the appropriate waza `Validator`
4. Outputs a JSON verdict in MSBench's expected format

```bash
# grader-shim.sh (generated by waza export)
#!/bin/bash
# Runs waza graders inside MSBench container
waza-grader \
  --config graders.yaml \
  --output "$MSBENCH_OUTPUT_FILE" \
  --fixtures "$MSBENCH_FIXTURES_DIR" \
  --format msbench-json
```

### Which Graders Travel Well?

| Grader | Portability | Notes |
|--------|-------------|-------|
| `keyword` | ✅ Easy | Pure string matching, no dependencies |
| `regex` | ✅ Easy | Standard regex, Go stdlib |
| `code` | ✅ Easy | Python assertions, needs Python runtime |
| `file` | ✅ Easy | File existence/content checks |
| `diff` | ✅ Easy | Golden file comparison |
| `json_schema` | ✅ Easy | Schema validation, pure logic |
| `program` | ⚠️ Medium | External scripts — need to be in container |
| `inline_script` | ⚠️ Medium | Embedded scripts — runtime must be available |
| `prompt` | ⚠️ Medium | LLM-as-judge — needs API access from container |
| `behavior` | ⚠️ Medium | Needs trajectory data in expected format |
| `action_sequence` | ⚠️ Medium | Same — trajectory-dependent |
| `skill_invocation` | 🔴 Hard | Tightly coupled to Copilot SDK session model |

**Phase 1 target:** Ship the shim with `keyword`, `regex`, `code`, `file`, `diff`, `json_schema` (the "pure" graders). These cover ~80% of real-world eval assertions.

**Phase 2:** Add `program`, `inline_script`, `prompt` support (needs container config for runtimes and API access).

### Shared Grader Format (Future)

If MSBench's Grader SDK v0 lands with a stable protocol, we should consider:

```yaml
# Universal grader descriptor
grader:
  type: waza/keyword          # namespaced type
  version: 0.8.0
  runtime: waza-grader-bin    # binary or container image
  config:
    keywords: ["recursion", "base case"]
    mode: all
  protocol: json-verdict      # stdin/stdout JSON contract
```

This is the long-term play: a grader interchange format that both platforms understand. But it requires MSBench's SDK to stabilize first.

### Effort Estimate

- Grader shim binary: 1 dev / 2 weeks
- `waza export` grader packaging: 1 dev / 1 week
- Phase 1 graders (6 types): included above
- Phase 2 graders (3 types): 1 dev / 2 weeks additional
- Total Phase 1: 1 dev / 3 weeks

---

## 5. What waza Brings to MSBench

These are concrete capabilities MSBench doesn't have today that waza could provide.

### 🏆 Trajectory Explorer

**MSBench status:** MISSING (planned 2 dev/2 months)
**waza status:** Shipped. Aspire-style waterfall timeline, event-by-event tool trace, session digest, expandable details.

**Integration paths:**
1. **Embed waza dashboard components** — The React `WaterfallTimeline`, `TrajectoryViewer`, and `TaskTrajectoryCompare` components could be packaged as an embeddable widget for MSBench's web UI
2. **waza serve as trajectory viewer** — `waza import --from msbench --run-id X && waza serve` gives MSBench users trajectory exploration immediately, no MSBench UI changes required
3. **Standalone trajectory renderer** — Extract trajectory rendering into a standalone npm package that either platform can use

**Recommendation:** Path 2 first (zero MSBench changes), path 1 later if there's appetite for deeper integration.

### 🏆 Decoupled Grading Platform

**MSBench status:** WORSE — metrics coupled to agent, building Grader SDK v0
**waza status:** 11 types, fully decoupled via `Validator` interface

**Integration path:** The grader shim (§4) directly provides this. MSBench teams get waza's grader library without changing their platform. They can even use `waza run --graders-only` to re-grade existing MSBench outputs locally.

### 🏆 Skill Development Lifecycle

**MSBench status:** No equivalent — MSBench measures quality after the skill exists
**waza status:** Full lifecycle: `init` → `new` → `suggest` → `dev` → `check` → `tokens` → `run`

**Integration path:** waza handles the authoring loop. When a skill is "ready" (passes `waza check`), it graduates to MSBench for scale validation. The `waza export` bridge is the handoff point.

### 🏆 Local Iteration Speed

**MSBench status:** Minutes to hours per run (container spin-up, queue, compute)
**waza status:** Seconds with mock executor, minutes with copilot-sdk

**Integration path:** This is the two-loop model itself. Fast inner loop in waza, slow outer loop in MSBench. The same eval definition powers both.

### 🏆 MCP Server for Conversational Eval

**MSBench status:** No equivalent
**waza status:** `waza dev --copilot` provides an MCP server for conversational skill development

**Integration path:** Future — once MSBench results flow into waza, the MCP server could provide conversational analysis of scale-run results. "Show me the failures from the last MSBench run that hit timeout errors."

---

## 6. What MSBench Brings to waza

### 🚀 Scale-Out Compute (100× Runs)

**waza limitation:** Running 100 trials on a laptop takes hours. CI runners have time limits.
**MSBench provides:** Distributed compute fleet that can run 100 instances in parallel.

**Usage pattern:** Skill author runs 3 trials locally (inner loop), then kicks off 100 runs via MSBench (outer loop) for statistical confidence before shipping.

### 🚀 Containerized Environments (Harbor Format)

**waza limitation:** Tasks run in temp directories with copied fixtures. No real toolchains, no real repos, no real file systems.
**MSBench provides:** Full containerized environments with real repos, real build systems, real test suites. Harbor-format benchmarks on Windows and Linux.

**Usage pattern:** waza's `code` and `file` graders work on simplified fixtures. MSBench environments let you test against real codebases. A skill that passes waza's "does it generate valid code?" might fail MSBench's "does the code actually compile and pass tests?"

### 🚀 Kusto Data Layer

**waza limitation:** Results are JSON files. No query engine, no cross-run aggregation, no ad-hoc analytics.
**MSBench provides:** Kusto DB with KQL for arbitrary queries. Custom KQL functions for benchmark analytics.

**Usage pattern:** For deep analysis (e.g., "what percentage of timeout failures occur in Python repos vs. TypeScript repos?"), query Kusto directly. For everyday visualization, `waza import` brings the data into the dashboard.

### 🚀 Telemetry-Driven Eval Sets

**waza limitation:** Eval tasks are hand-authored by skill developers.
**MSBench provides:** Real user failure cohorts derived from production telemetry. "These are the 50 prompts that failed most often last week."

**Usage pattern:** MSBench identifies failure patterns from production → exports as waza task files → skill author uses them for targeted development. This closes the loop between production quality and skill authoring.

### 🚀 Production-Like Agent Environments

**waza limitation:** Mock executor or copilot-sdk on local machine.
**MSBench provides:** CES CAPI proxy, special agents shim, production-equivalent tool access.

**Usage pattern:** An eval that passes with `executor: copilot-sdk` locally might behave differently in production due to rate limits, model routing, or tool availability. MSBench provides the production-equivalent execution path.

---

## 7. Architecture Options

### Option A: waza as MSBench Frontend (Thin Adapter)

```
User ──► waza CLI ──► MSBench API ──► Compute Fleet
                  ◄── Results ◄──── Kusto
```

**What it means:** waza becomes a client for MSBench. `waza run --backend msbench` sends evals to MSBench for execution instead of running locally. Results stream back into waza's dashboard.

**Pros:**
- Single tool for the skill author (waza is the only CLI they touch)
- Unified dashboard for local and cloud results
- Lowest friction for users

**Cons:**
- Deep coupling — waza depends on MSBench API stability
- Auth complexity — waza needs to handle Entra ID, Kusto connections
- MSBench team has to support an external consumer (API contract)
- Blurs ownership: who owns the execution pipeline?

**Effort:** 2-3 dev / 8-10 weeks
**Risk:** High — requires close MSBench team collaboration and API stability

### Option B: Shared Eval Format with Bidirectional Sync

```
                    ┌───────────────┐
waza CLI ──export──►│  Shared Eval  │◄──import── MSBench CLI
waza CLI ◄─import───│  Format (SEF) │──export──► MSBench CLI
                    └───────────────┘
```

**What it means:** Define a "Shared Eval Format" that both tools can read and write. Neither depends on the other's API. The format is the contract.

**Pros:**
- Loose coupling — each tool evolves independently
- Clear contract: the format spec is the only shared artifact
- Either side can adopt incrementally
- No runtime dependency between platforms

**Cons:**
- Format design is hard — two different execution models (local vs. containerized)
- Bidirectional sync means maintaining two converters
- Risk of format drift as both platforms evolve
- Grader portability is still a hard problem

**Effort:** 2 dev / 6-8 weeks (format design + both converters)
**Risk:** Medium — format negotiation takes time, but low runtime risk

### Option C: waza Generates MSBench Benchmarks (One-Way Export)

```
waza CLI ──export──► MSBench Benchmark Package ──► MSBench CLI
waza CLI ◄─import─── MSBench Results (Kusto/JSON)
```

**What it means:** waza exports to MSBench format. MSBench results can be imported back. But MSBench never writes waza format. The flow is: author in waza → run at scale in MSBench → review in waza.

**Pros:**
- **Simplest to build** — one converter direction for evals, one for results
- Matches the natural workflow (author locally → validate at scale)
- waza stays the authoring tool, MSBench stays the compute platform
- No shared format negotiation needed
- Can ship incrementally (export first, import later)

**Cons:**
- MSBench-authored benchmarks can't flow into waza (one-way only)
- Doesn't help MSBench teams who don't use waza for authoring
- Less symmetric than Option B

**Effort:** 1 dev / 4-5 weeks
**Risk:** Low — minimal coupling, clear ownership boundaries

### Recommendation: Option C, with a Path to B

**Start with Option C** because:
1. It matches the actual workflow (inner loop → outer loop)
2. It's the smallest investment with the most immediate value
3. It doesn't require MSBench to change anything — we adapt to them
4. It can ship in phases: `waza export` (week 1-3) → `waza import` (week 3-5)

**Evolve toward Option B** if:
- MSBench teams want to use waza's graders natively
- A shared eval format emerges from real usage patterns
- There's demand for MSBench → waza authoring flow

**Never go to Option A** unless MSBench exposes a stable public API and there's organizational commitment to maintain it.

---

## Implementation Roadmap

### Phase 1: Export + Grader Shim (Weeks 1-5)

| Week | Deliverable |
|------|-------------|
| 1-2 | `waza export --target msbench` — generates benchmark.yaml + instance dirs |
| 2-3 | Grader shim binary — packages 6 "pure" graders as MSBench-callable scripts |
| 3-4 | `waza export` integration testing with a real MSBench benchmark run |
| 4-5 | Documentation + example: end-to-end inner → outer loop walkthrough |

**Exit criteria:** A skill author can run `waza export`, hand the output to MSBench, and get a valid 100-run benchmark.

### Phase 2: Import + Unified Dashboard (Weeks 5-9)

| Week | Deliverable |
|------|-------------|
| 5-6 | `waza import --from msbench --run-id <id>` — Kusto → waza JSON |
| 6-7 | Dashboard: `source` tag on results, MSBench runs visible in TrendsPage |
| 7-8 | Trajectory format bridge (ATIF → waza format, if ATIF available) |
| 8-9 | CompareView: compare local run (3 trials) vs. MSBench run (100 trials) |

**Exit criteria:** A skill author can see MSBench results in waza's dashboard alongside local results.

### Phase 3: Deep Integration (Weeks 9-14)

| Week | Deliverable |
|------|-------------|
| 9-10 | Phase 2 graders in shim (`program`, `inline_script`, `prompt`) |
| 10-11 | Telemetry-driven task generation: MSBench failure cohorts → waza task files |
| 11-12 | `waza run --graders-only` — re-grade MSBench outputs with waza's full grader set |
| 12-14 | Trajectory explorer as embeddable widget (optional, depends on MSBench appetite) |

**Exit criteria:** Full two-loop workflow with grader portability and trajectory visualization across both platforms.

---

## 8. MSBench Benchmark Format (Azure CLI Reference)

Based on Shayne's intel from the MSBench wiki and Azure DevOps repos, here's what we learned about MSBench's containerized approach:

### MSBench Repository Structure

**Location:** `msbench-benchmarks` repo in Azure DevOps
**Organization:** Benchmarks live under `curation/benchmarks/{product}/`

Example from Azure CLI:
```
msbench-benchmarks (Azure DevOps repo)
├── curation/
│   └── benchmarks/
│       ├── azure/                    # Azure CLI benchmarks
│       │   ├── benchmark.yaml        # Benchmark manifest
│       │   ├── instances/
│       │   │   ├── create-resource/
│       │   │   │   ├── Dockerfile    # Containerized test environment
│       │   │   │   ├── instance.yaml
│       │   │   │   └── fixtures/
│       │   │   └── ...
│       │   └── grader-shim.sh       # Grader entry point
│       ├── other-product/
│       └── ...
```

### Azure CLI Containerization Model

The Azure CLI benchmarks use **Docker containers to package**:
1. The Azure CLI binary (or from source)
2. Test environment (Python, shell, required tools)
3. Golden fixtures and test suites
4. Grading logic (as executable scripts)

**Container flow:**
```
Dockerfile ──build──► Harbor Image ──deploy──► MSBench Compute Fleet
                                                (50-100 parallel runs)
                                                │
                                                ├─► Container instance 1
                                                ├─► Container instance 2
                                                └─► Container instance N
                                                     (all run same test suite)
```

Each container instance:
- Executes an isolated task (e.g., "test `az storage create` command")
- Captures output (stdout, stderr, exit code)
- Runs grader logic (in-container)
- Writes results to Kusto (via MSBench sidecar)

### Implications for waza Export

When we implement `waza export --target msbench`, the export flow must generate **containerized benchmark packages**:

```
waza export --target msbench --output bench/
```

Produces:
```
bench/
├── benchmark.yaml              # MSBench benchmark manifest
├── instances/
│   ├── task-001/
│   │   ├── Dockerfile          # ✨ NEW: Container spec
│   │   ├── instance.yaml       # MSBench instance config
│   │   ├── prompt.md           # Task description
│   │   ├── fixtures/           # Test data
│   │   ├── grader-shim.sh      # Waza graders wrapped as executable
│   │   └── requirements.txt    # (if Python graders needed)
│   └── task-002/
│       └── ...
└── docker-compose.yaml         # (optional) For local testing
```

**The Dockerfile template** (generated by waza):
```dockerfile
FROM mcr.microsoft.com/azure-cli:latest
# or FROM python:3.11, FROM node:20, etc. depending on eval needs

# Install waza grader runtime
COPY waza-grader /usr/local/bin/

# Set up eval environment
WORKDIR /eval
COPY fixtures/ ./fixtures/
COPY grader-shim.sh ./
RUN chmod +x ./grader-shim.sh

# Entry point: run the agent, then run graders
ENTRYPOINT ["/bin/bash", "-c", "waza-grader --config graders.yaml --output /tmp/result.json && cat /tmp/result.json"]
```

**What this means for waza's export logic:**
1. Infer base image from eval config (e.g., `config.base_image: python:3.11` or auto-detect from graders)
2. Embed waza grader binary in each Dockerfile
3. Generate grader-shim.sh that MSBench's container runtime can invoke
4. Package fixtures + eval config into instance directory
5. Return a complete, buildable benchmark package

### Results Flow Through Containers

```
Container runs ──► stdout/stderr ──► MSBench sidecar ──► Kusto
                                                           │
                                   waza import ◄──────────┘
                                   (extract via KQL)
                                        │
                                        ▼
                                  results.json
                                  (waza format)
```

### Reference Links

- **MSBench Wiki** (internal, EMU auth): https://github.com/devdiv-microsoft/MicrosoftSweBench/wiki
  - Authoring guides for benchmark specs
  - Grader documentation (SDK v0)
  - Container image best practices
  - Harbor format reference

- **Azure CLI Benchmarks** (Azure DevOps): https://dev.azure.com/devdiv/OnlineServices/_git/msbench-benchmarks?path=/curation/benchmarks/azure
  - Real-world example of containerized benchmark structure
  - Dockerfile patterns and base images used
  - Instance configuration for large-scale runs

- **MSBench Team**: For deep dives on Dockerfile requirements, Harbor pipeline, and Kusto schema

### Key Takeaway

The "containerized" model is not incidental — it's **core to MSBench's scale story**. Every run is isolated, reproducible, and self-contained. waza's `export` command must generate valid Dockerfiles if we want benchmarks to run reliably at scale.

**First iteration approach:**
- Generate a minimal, opinionated Dockerfile template
- Support "standard" base images (Python, Node, Go, Azure CLI, .NET)
- Allow expert users to override via `config.dockerfile` in eval.yaml
- Test against MSBench's Harbor pipeline early to validate format compatibility

---

## 10. Harbor Runtime — MSBench's Container Direction

MSBench is transitioning to **Harbor** (https://github.com/laude-institute/harbor), an open-source framework from the Laude Institute (creators of Terminal-Bench) for running agent evaluations at scale. This shifts the container architecture significantly and changes how waza should integrate.

### What Harbor Is

- **Open-source framework** from Laude Institute for containerized agent evaluation benchmarks
- **GitHub:** https://github.com/laude-institute/harbor
- **Architecture:** Each benchmark task runs in isolated Docker containers managed by Harbor
- **Horizontal scale:** Orchestrates containers across cloud providers (Daytona, Modal, AWS, Azure) or locally
- **Multi-dimensional metrics:** Accuracy, cost, test pass rate, latency, reliability, agent efficiency
- **SWE-bench pedigree:** Built for software engineering evals — real-world tasks in sandboxed containers (like Terminal-Bench)
- **CLI:** `harbor run --dataset terminal-bench@2.0 --agent claude-code --model anthropic/claude-opus-4-1 --n-concurrent 4`

### What Harbor Means for waza × MSBench Integration

This changes the integration model in a critical way:

**Instead of waza generating Dockerfiles directly, waza should target Harbor benchmark format.**

This is a cleaner integration point:

1. **`waza export --target harbor`** — Generate a Harbor-compatible benchmark definition (not a Dockerfile)
   - Harbor handles all containerization, isolation, and lifecycle
   - waza stays focused on eval authoring, local execution, and results analysis
   - One-way export: waza evals → Harbor benchmarks

2. **Results flow:** Harbor produces structured results (JSON, metrics)
   - Map Harbor results back to waza's `results.json` format via `waza import --from harbor`
   - Bridge enables trajectory tracking and comparison in waza dashboard

3. **Grader compatibility:** Harbor supports custom grader plugins
   - waza's grader binaries can run as Harbor graders
   - Means one grader implementation works in both contexts

4. **Key architectural insight:** The abstraction layer is **Harbor**, not raw Docker
   - waza generates Harbor configs, not Dockerfiles
   - Harbor + MSBench handle orchestration, scale, cloud placement
   - This is more resilient to MSBench/Harbor internals changes

### Updated Architecture with Harbor

```
Inner Loop (waza)                    Outer Loop (MSBench + Harbor)
┌─────────────────┐                  ┌──────────────────────────────┐
│ waza init/new    │                  │ Harbor Runtime               │
│ waza suggest     │                  │ ┌──────────────────────────┐ │
│ waza run (1-5x)  │ waza export     │ │ Container Fleet          │ │
│ waza serve       │ ──────────────→ │ │ • 50-100 parallel tasks  │ │
│ waza dev         │ --target harbor │ │ • Docker isolation       │ │
│                  │                  │ │ • Cloud orchestration    │ │
│ Results ←────────│ waza import     │ │   (Daytona/Modal/Azure)  │ │
│ Dashboard  ←─────│ ←────────────── │ │ • Real-world environments│ │
│ Trajectory View  │ --from harbor   │ └──────────────────────────┘ │
│ Compare Runs     │                  │                              │
│                  │                  │ MSBench Compute (CES)        │
│                  │                  │ Results → Kusto Analytics    │
└─────────────────┘                  └──────────────────────────────┘
```

### Harbor Integration Roadmap

**Phase 1:** Export to Harbor format
- [ ] Study Harbor benchmark schema (`.harbor/benchmark.yaml` format)
- [ ] Implement `waza export --target harbor` (converts eval.yaml → Harbor benchmark config)
- [ ] Test against Harbor CLI locally (`harbor run --benchmark waza-export/`)
- [ ] Validate grader compatibility

**Phase 2:** Results import + trajectory tracking
- [ ] Implement `waza import --from harbor` (Harbor results JSON → waza results.json)
- [ ] Map Harbor metrics to waza dimensions (accuracy, cost, latency, etc.)
- [ ] Add trajectory comparison for cross-platform runs (waza local vs. Harbor cloud)

**Phase 3:** Grader plugin registration
- [ ] Package waza graders as Harbor plugins
- [ ] Allow Harbor benchmark authors to reference waza graders by name

### Reference Links

- **MSBench source:** https://dev.azure.com/devdiv/InternalTools/_git/MicrosoftSweBench
- **MSBench benchmarks:** https://dev.azure.com/devdiv/OnlineServices/_git/msbench-benchmarks
- **MSBench wiki:** https://github.com/devdiv-microsoft/MicrosoftSweBench/wiki
- **Harbor framework:** https://github.com/laude-institute/harbor
- **Harbor evaluation benchmarks:** https://deepwiki.com/harbor-framework/awesome-harbor/3-evaluation-benchmarks
- **Terminal-Bench (Harbor precedent):** https://github.com/laude-institute/terminal-bench

---

## 9. Open Questions for MSBench Team

1. **Exact Dockerfile/benchmark spec format** — What's the schema for `instance.yaml` and `benchmark.yaml`? Are there examples in a public wiki page or repo we can reference?

2. **Results schema and Kusto ingestion** — How do container exit codes, stdout, stderr, and grader outputs map to Kusto columns? What's the table schema for benchmark results? Can we get sample KQL queries?

3. **Read access to `msbench-benchmarks`** — Can we get read access to the Azure DevOps repo (specifically `curation/benchmarks/azure/`) to study the Azure CLI benchmark structure firsthand? This would help us design waza's export format to be compatible.

4. **Benchmark authoring guide** — Is there a wiki page or document that documents the benchmark spec format, Dockerfile requirements, and best practices? This would be the reference for waza's export logic.

5. **Grader SDK v0 interface** — What's the stable protocol for MSBench graders? stdin/stdout JSON? gRPC? HTTP? File-based? This determines how we package waza's grader shim. Does the wiki have a gRPC schema or interface spec?

6. **Container runtime and base images** — What base images does MSBench pull from (MCR, Docker Hub, internal registry)? Are there image size or build time constraints we should know about? Can we add binaries like `waza-grader` to existing images, or do we need to build our own?

7. **Can waza's embedded dashboard serve as the trajectory explorer?** — MSBench is currently missing trajectory visualization (planned 2 dev/2 months). Once we implement `waza import`, would embedding the waza dashboard's trajectory viewer be valuable for MSBench users, or are you building a native MSBench solution?

### Harbor-Specific Questions

8. **Harbor migration timeline** — Are you on Harbor already, or in transition? What's the ETA for MSBench to be fully Harbor-based? This affects our integration sequencing.

9. **Harbor benchmark format/schema** — What's the exact `benchmark.yaml` (or `.harbor/config`) format Harbor expects? Where are example benchmarks? We need the spec to implement `waza export --target harbor` accurately.

10. **Harbor grader plugin interface** — Can waza register as a "grader plugin" in Harbor's framework? What's the plugin protocol (how does Harbor invoke custom graders)? Can we deploy waza graders to Harbor's plugin registry?

11. **Harbor results API** — Does Harbor have a programmatic results API we can pull from? We need this for `waza import --from harbor` to fetch benchmark results and map them back to waza's results format. Is it REST, gRPC, or file-based?

12. **Harbor dataset/benchmark publishing** — Can we publish waza exports as Harbor datasets or benchmarks? Do you have a registry or artifact store for community benchmarks? This would let other teams discover and run waza-authored evals on Harbor.

---

## 11. Concrete Format Mapping: waza eval.yaml → MSBench Benchmark

Now that we have the exact Azure CLI benchmark structure from the MSBench ADO repo, we can stop speculating and define the precise field-level mapping. This section replaces the earlier hypothetical Harbor/Dockerfile guesses with ground truth.

### A. Field-by-Field Mapping

#### Top-Level: eval.yaml → benchmark_metadata.json

| waza Field (eval.yaml) | MSBench Equivalent | Transformation | Notes |
|---|---|---|---|
| `name` (e.g. `code-explainer-eval`) | `instance_id` prefix | Slugify → `code-explainer-eval-{task_slug}` | MSBench uses flat instance IDs, not nested. The eval name becomes a namespace prefix for all task instance IDs |
| `description` | Not in metadata JSON | Stored in `version.txt` comment or README | MSBench metadata is minimal — description lives outside the JSON |
| `version` | `version.txt` | `msbench-{waza_version}` (e.g. `msbench-0.8.0`) | MSBench versions the benchmark, not individual tasks |
| `skill` | N/A | Informational only | MSBench doesn't have a skill concept — this is waza metadata carried as a comment |
| `config.trials_per_task` | `--n` flag on `msbench-cli run` | Not in benchmark package — runtime config | MSBench controls repetition at the CLI level, not in metadata |
| `config.timeout_seconds` | No direct equivalent | Encoded in `entry.sh` as a timeout wrapper | Could use `timeout {N}s` in the entrypoint script |
| `config.model` | Agent YAML (e.g. `github-copilot-cli_models.yaml`) | Separate config file | MSBench decouples model selection from benchmark definition |
| `config.executor` | `--agent` flag on `msbench-cli run` | Maps to agent name (e.g. `github-copilot-cli`) | waza executor = MSBench agent |
| `config.parallel` | MSBench runtime handles parallelism | Not in benchmark package | MSBench always runs instances in parallel across fleet |
| `metrics[]` | Not in MSBench | Dropped (MSBench has its own aggregation in Kusto) | waza metrics are a local concept — MSBench uses Kusto-side analytics |

#### Per-Task: tasks/*.yaml → benchmark_metadata.json entries + per-instance packages

| waza Field (task YAML) | MSBench Equivalent | Transformation | Notes |
|---|---|---|---|
| `id` (e.g. `explain-python-recursion-001`) | `instance_id` | `{eval_name}-{task_id}` → `code-explainer-eval-explain-python-recursion-001` | Must be globally unique across all MSBench benchmarks |
| `name` | Informational only | Carried in package metadata | MSBench identifies by `instance_id`, not display name |
| `description` | Part of `problem_statement` | Prepended to the problem statement | Gives the agent context about what the task is testing |
| `inputs.prompt` | `problem_statement` | Direct mapping — the core instruction the agent receives | This is the single most important field. 1:1 mapping |
| `inputs.files[]` | `setup/setup_{instance_id}.sh` | Script that copies fixture files into `/testbed/` | MSBench uses setup scripts to prepare the workspace, not static file references |
| `inputs.context` | Encoded in `problem_statement` or setup script | Metadata like `language: python` appended to prompt or set as env vars | No structured context in MSBench — must be flattened |
| `inputs.environment` | `activation_script` in package | `export KEY=VALUE` lines in the activation script | Environment variables set before agent runs |
| `expected.output_contains[]` | `eval_script` (assertions) | Becomes Python `assert` statements in `eval/eval_{instance_id}.py` | waza's `MustInclude` → `assert "keyword" in output` |
| `expected.output_not_contains[]` | `eval_script` (negative assertions) | `assert "bad_keyword" not in output` | waza's `MustExclude` → negative Python asserts |
| `expected.behavior` | Not directly supported | Requires custom eval script logic | MSBench eval scripts could check tool call count, but it's not standard |
| `graders[]` | `eval_script` + `parse_{name}.py` | Each grader compiled to Python eval/parse script pair | See §11C below for the grader bridge |
| `tags[]` | Not in MSBench | Could be encoded in instance_id suffix or metadata CSV columns | MSBench doesn't have a tagging system — filtering is by instance_id pattern |
| `timeout_seconds` (per-task) | Timeout in `entry.sh` | `timeout {N}s` wrapping the agent invocation | Per-task timeout override |
| `enabled` | Include/exclude from `benchmark_metadata.json` | Disabled tasks simply not exported | No concept of disabled instances in MSBench |

#### Grader Config: graders[] → eval scripts

| waza Grader Field | MSBench Equivalent | Notes |
|---|---|---|
| `type` (GraderKind) | Determines which eval template to use | See grader bridge table in §11C |
| `name` | eval script function/file name | `eval_{grader_name}.py` or function name within consolidated eval script |
| `config.assertions[]` | Python assert statements | Direct transpilation for `code` grader |
| `config.must_match[]` / `must_not_match[]` | `re.search()` / `not re.search()` | Regex grader → Python `re` module calls |
| `config.keywords[]` | `in` operator checks | Keyword grader → `assert "word" in output` |
| `config.expected_file` | `diff` or `filecmp` call | Diff grader → golden file comparison |
| `config.schema` | `jsonschema.validate()` | JSON schema grader → Python jsonschema library |
| `weight` | Not in MSBench | MSBench eval scripts return pass/fail, not weighted scores |
| `script` (for program grader) | Copied into eval/ directory | External script bundled as-is |

### B. What `waza export --target msbench` Would Generate

```
output/
├── benchmark_metadata.json          # Generated from eval.yaml + all task YAMLs
├── version.txt                      # "msbench-{waza_version}" (e.g. msbench-0.8.0)
├── generate_metadata_csv.py         # Copied from MSBench template (generates metadata.csv)
├── prepare_metadata.py              # Copied from MSBench template (builds per-instance tar.gz)
├── secret_files.txt                 # Empty or user-provided
├── {agent}_models.yaml              # Generated from config.model (agent model config)
├── CHANGELOG.md                     # Auto-generated with export timestamp
│
└── docker/
    ├── Dockerfile                   # Generated from MSBench template (uses INSTANCE_ID build-arg)
    ├── vendor/                      # Must be provided by MSBench team (install_all.sh etc.)
    │
    ├── eval/                        # Compiled graders → Python eval scripts
    │   ├── eval_{instance_id_1}.py  # Eval script for task 1 (from waza graders)
    │   ├── eval_{instance_id_2}.py  # Eval script for task 2
    │   ├── parse_{instance_id_1}.py # Parse script for task 1 (extracts result from output)
    │   ├── parse_{instance_id_2}.py # Parse script for task 2
    │   └── waza_grader_lib.py       # Shared helper library (regex, keyword, diff utils)
    │
    ├── setup/                       # Workspace setup from context_files/fixtures
    │   ├── setup_{instance_id_1}.sh # Copies fixture files into /testbed/
    │   └── setup_{instance_id_2}.sh # Each task gets its own setup script
    │
    └── packages/                    # Generated by prepare_metadata.py (gitignored)
        ├── {instance_id_1}.tar.gz   # Per-instance bundle (entry.sh, activation, eval, setup)
        └── {instance_id_2}.tar.gz
```

#### Generated `benchmark_metadata.json`

From a waza eval like `code-explainer-eval` with two tasks:

```json
[
  {
    "instance_id": "code-explainer-eval-explain-python-recursion-001",
    "problem_statement": "Explain this code to me\n\n---\nContext: Python beginner-level code demonstrating recursion.\nSee factorial.py in your workspace.",
    "eval_script": "eval/eval_code-explainer-eval-explain-python-recursion-001.py",
    "setup_script": "setup/setup_code-explainer-eval-explain-python-recursion-001.sh",
    "parse_script": "eval/parse_code-explainer-eval-explain-python-recursion-001.py"
  },
  {
    "instance_id": "code-explainer-eval-explain-js-async-002",
    "problem_statement": "Explain this code to me\n\n---\nContext: JavaScript async/await pattern.\nSee async-fetch.js in your workspace.",
    "eval_script": "eval/eval_code-explainer-eval-explain-js-async-002.py",
    "setup_script": "setup/setup_code-explainer-eval-explain-js-async-002.sh",
    "parse_script": "eval/parse_code-explainer-eval-explain-js-async-002.py"
  }
]
```

#### Generated `entry.sh` (per-instance, packed in tar.gz)

```bash
#!/bin/bash
set -e

# Activate environment (set env vars, paths)
source /activation_script.sh

# Setup workspace (copy fixtures into /testbed)
bash /setup_script.sh

# Agent execution happens here (MSBench runtime injects the agent)
# The agent reads problem_statement and works in /testbed

# After agent completes, run evaluation
cd /testbed
python3 /eval_script.py /testbed /output/eval_result.json

# Parse results into MSBench format
python3 /parse_script.py /output/eval_result.json /output/result.json
```

#### Generated eval script example (for a task with `code` + `regex` + `keyword` graders)

```python
#!/usr/bin/env python3
"""Auto-generated by waza export --target msbench
Source: code-explainer-eval / explain-python-recursion-001
Graders: has_explanation (code), no_errors (regex), explains_recursion (code)
"""
import json
import re
import sys
import os

def read_output(testbed_dir):
    """Read agent output from testbed."""
    # MSBench convention: agent output captured to /output/agent_output.txt
    output_file = os.path.join("/output", "agent_output.txt")
    if os.path.exists(output_file):
        with open(output_file) as f:
            return f.read()
    return ""

def grade(testbed_dir, output_dir):
    output = read_output(testbed_dir)
    results = []

    # Grader: has_explanation (type: code)
    # Source assertions: ["len(output) > 10"]
    try:
        assert len(output) > 10, "Output too short"
        results.append({"name": "has_explanation", "passed": True, "score": 1.0})
    except AssertionError as e:
        results.append({"name": "has_explanation", "passed": False, "score": 0.0, "error": str(e)})

    # Grader: no_errors (type: regex, must_not_match)
    # Patterns: ["(?i)fatal error|crashed|exception occurred"]
    error_patterns = [r"(?i)fatal error|crashed|exception occurred"]
    regex_passed = True
    for pattern in error_patterns:
        if re.search(pattern, output):
            regex_passed = False
            break
    results.append({"name": "no_errors", "passed": regex_passed, "score": 1.0 if regex_passed else 0.0})

    # Grader: explains_recursion (type: code)
    # Source assertions: ["len(output) > 10"]
    try:
        assert len(output) > 10, "Output too short for recursion explanation"
        results.append({"name": "explains_recursion", "passed": True, "score": 1.0})
    except AssertionError as e:
        results.append({"name": "explains_recursion", "passed": False, "score": 0.0, "error": str(e)})

    # expected.output_contains: ["recursive", "factorial"]
    for keyword in ["recursive", "factorial"]:
        found = keyword.lower() in output.lower()
        results.append({"name": f"contains_{keyword}", "passed": found, "score": 1.0 if found else 0.0})

    # Aggregate: pass if all graders pass
    all_passed = all(r["passed"] for r in results)

    verdict = {
        "instance_id": "code-explainer-eval-explain-python-recursion-001",
        "passed": all_passed,
        "score": sum(r["score"] for r in results) / len(results),
        "graders": results,
    }

    os.makedirs(output_dir, exist_ok=True)
    with open(os.path.join(output_dir, "eval_result.json"), "w") as f:
        json.dump(verdict, f, indent=2)

    return verdict

if __name__ == "__main__":
    testbed = sys.argv[1] if len(sys.argv) > 1 else "/testbed"
    output = sys.argv[2] if len(sys.argv) > 2 else "/output"
    result = grade(testbed, output)
    sys.exit(0 if result["passed"] else 1)
```

#### Generated setup script example

```bash
#!/bin/bash
# Auto-generated by waza export --target msbench
# Instance: code-explainer-eval-explain-python-recursion-001
# Source fixtures: examples/code-explainer/fixtures/

set -e

mkdir -p /testbed

# Copy fixture files into workspace
cat > /testbed/factorial.py << 'WAZA_EOF'
def factorial(n):
    if n <= 1:
        return 1
    return n * factorial(n - 1)
WAZA_EOF

echo "Workspace ready: /testbed"
ls -la /testbed/
```

#### Generated `metadata.csv` columns (from `generate_metadata_csv.py`)

```csv
instance_id,docker_image_tag
code-explainer-eval-explain-python-recursion-001,azure.eval.x86_64.code-explainer-eval-explain-python-recursion-001
code-explainer-eval-explain-js-async-002,azure.eval.x86_64.code-explainer-eval-explain-js-async-002
```

### C. The Grader Bridge: waza Validators → MSBench eval_script

Each waza grader type compiles to a Python function in the generated eval script. Here's the exact mapping:

| waza Grader | MSBench eval_script Translation | Complexity | Runtime Deps | Example |
|---|---|---|---|---|
| `code` | Python `assert` statements (direct transpile from `config.assertions[]`) | 🟢 Trivial | None | `assert len(output) > 10` |
| `regex` | `re.search()` / `re.match()` calls using `config.must_match[]` and `config.must_not_match[]` | 🟢 Trivial | `re` (stdlib) | `assert re.search(r"def \w+", output)` |
| `keyword` | Python `in` operator for `config.keywords[]` with `mode: all\|any` logic | 🟢 Trivial | None | `assert "recursion" in output.lower()` |
| `file` | `os.path.exists()` + `open().read()` for file content checks in `/testbed/` | 🟢 Easy | `os` (stdlib) | `assert os.path.exists("/testbed/output.py")` |
| `diff` | `filecmp.cmp()` or line-by-line diff against golden file bundled in package | 🟢 Easy | `filecmp` (stdlib) | `assert filecmp.cmp("/testbed/out.py", "/golden/out.py")` |
| `json_schema` | `jsonschema.validate()` with schema from `config.schema` | 🟡 Medium | `jsonschema` (pip) | `jsonschema.validate(json.loads(output), schema)` |
| `program` | External script copied into `eval/` directory, invoked via `subprocess.run()` | 🟡 Medium | Depends on script | `subprocess.run(["python3", "eval/custom_check.py"])` |
| `inline_script` | Not used (waza's `code` grader covers this) | N/A | N/A | — |
| `prompt` | 🔴 **Cannot run in standard MSBench container** — needs LLM API access | 🔴 Hard | API key + network | Would need `AZURE_OPENAI_ENDPOINT` env var in container |
| `behavior` | Custom logic checking `SessionDigest` fields (tool count, token usage) | 🟡 Medium | Trajectory JSON | `assert tool_call_count <= 5` |
| `action_sequence` | Ordered tool call sequence validation against trajectory | 🟡 Medium | Trajectory JSON | `assert tools_used == ["read_file", "edit_file"]` |
| `skill_invocation` | 🔴 **Not portable** — tightly coupled to Copilot SDK session model | 🔴 Not supported | Copilot SDK | Export warns and skips |

**Phase 1 export target (MVP):** `code`, `regex`, `keyword`, `file`, `diff` — these 5 cover ~80% of real evals and compile to pure Python with zero external deps.

**Phase 2:** Add `json_schema` (needs pip install in Dockerfile), `program` (bundle scripts), `behavior` + `action_sequence` (need trajectory data format from MSBench).

**Not portable:** `prompt` (needs LLM API in container) and `skill_invocation` (Copilot-specific). Export emits a warning and skips these graders with a comment in the eval script.

### D. What We Need From the MSBench Team

To build `waza export --target msbench`, we need these specific artifacts:

| Artifact | Why We Need It | Blocking? |
|---|---|---|
| **`benchmark_metadata.json` schema** — Exact field names, types, required vs optional. Is it a JSON array of objects? Is `eval_script` a path or inline? Are there fields beyond `instance_id`, `problem_statement`, `eval_script`? | Without the schema, our generated JSON may be rejected by the pipeline | 🔴 Yes |
| **`entry.sh` template** — The entrypoint script that gets packed into per-instance tar.gz. What does it invoke? What env vars does it expect? How does the agent get injected? | We need to generate compatible entrypoints | 🔴 Yes |
| **`vendor/install_all.sh`** — Or documentation on what it installs (Python version, pip packages, Node version, system deps) | We need to know what's available in the container at eval-script runtime | 🟡 Partial (can infer from Dockerfile) |
| **`prepare_metadata.py` source** — The script that builds per-instance tar.gz packages. What goes in each tarball? Directory layout inside the tar? | We need to generate compatible tar.gz packages, or rely on their script | 🟡 Partial (can reverse-engineer) |
| **`metadata.csv` column format** — Exact columns. Is it just `instance_id,docker_image_tag` or are there more? | Our `generate_metadata_csv.py` must produce the right format | 🟡 Yes for CI |
| **Results output schema** — What goes in `/output/`? File names, JSON structure, exit code semantics (0 = pass, 1 = fail?) | Our eval scripts must write results in the expected format | 🔴 Yes |
| **`activation_script` format** — What does the per-instance activation script do? Env vars? Path setup? | Need to generate from waza's `inputs.environment` | 🟡 Partial |
| **ACR push permissions / registry path** — Do we push our own images, or does the MSBench pipeline build them? | Determines if `waza export` needs a `--push` flag or just generates source | 🟢 Nice to have |
| **Agent injection mechanism** — How does the agent (e.g. Copilot CLI) get invoked inside the container? Does MSBench inject it, or does `entry.sh` call it? | Critical for understanding the eval flow | 🔴 Yes |
| **Sample tar.gz contents** — One real per-instance tar.gz from the Azure CLI benchmark so we can see the exact file layout | Removes all guesswork about the package format | 🔴 Yes |

### E. Prototype Scope: Minimal Viable `waza export --target msbench`

The simplest thing that could work — enough to hand the MSBench team a real artifact and validate the format.

#### What the MVP Does

```bash
waza export --target msbench -o ./msbench-export/ examples/code-explainer/eval.yaml
```

1. **Reads** eval.yaml + all referenced task YAMLs + fixture files
2. **Generates** `benchmark_metadata.json` with one entry per task
3. **Compiles** Phase 1 graders (`code`, `regex`, `keyword`, `file`, `diff`) to Python eval scripts
4. **Creates** setup scripts that embed fixture file contents
5. **Copies** Dockerfile and helper script templates
6. **Outputs** a directory that the MSBench team can `prepare_metadata.py` → `build_images.sh` → `msbench-cli run`

#### What the MVP Does NOT Do

- No `prompt`, `behavior`, `action_sequence`, or `skill_invocation` grader support (warns and skips)
- No `json_schema` support (needs pip dep in container — Phase 2)
- No Docker image building (MSBench pipeline does that)
- No ACR pushing
- No results import (`waza import --from msbench` is separate work)
- No `vendor/` directory (must be provided by MSBench team or symlinked)
- No `prepare_metadata.py` / `generate_metadata_csv.py` (copied from MSBench templates)

#### MVP Effort Estimate

| Component | Effort | Notes |
|---|---|---|
| CLI command (`waza export --target msbench`) | 2 days | Flag parsing, output dir creation, orchestration |
| benchmark_metadata.json generator | 2 days | Walk eval.yaml → task YAMLs → JSON |
| Grader compiler (5 Phase 1 types → Python) | 3 days | Template-based code generation for each grader type |
| Setup script generator | 1 day | Embed fixtures as heredocs in bash scripts |
| Dockerfile + template files | 1 day | Static templates with INSTANCE_ID placeholder |
| Entry.sh generator | 1 day | Per-instance entrypoint from template |
| Tests | 2 days | Round-trip: eval.yaml → export → validate structure |
| **Total MVP** | **~2 weeks** | 1 dev, assumes MSBench team provides vendor/ and templates |

#### MVP Validation Plan

1. Export the `code-explainer-eval` example → MSBench format
2. Hand the output directory to the MSBench team
3. They run `prepare_metadata.py` + `build_images.sh` + `msbench-cli run`
4. If the container builds and the eval script runs → format is validated
5. Iterate on any field/path mismatches

This is the fastest path to a working handshake. We generate, they validate.

---

## What We're NOT Proposing

- **Replacing MSBench's UI** — MSBench's web app serves its users. We're adding waza's trajectory explorer as a complementary view, not a replacement.
- **Merging the platforms** — waza and MSBench have different users, different deployment models, different priorities. Integration ≠ merger.
- **Building a new standard** — We're adapting to MSBench's existing formats, not proposing a new industry standard for eval specs.
- **Running MSBench locally** — MSBench is a cloud platform. We're not trying to shrink it down to a laptop.

---

## Summary

| Dimension | Approach | Effort |
|-----------|----------|--------|
| Two-loop model | Author in waza, scale in MSBench | Workflow + docs |
| Eval format bridge | `waza export --target msbench` (one-way) | 1 dev / 2-3 weeks |
| Results flow | `waza import --from msbench` (Kusto → JSON) | 1 dev / 3-4 weeks |
| Grader compatibility | Grader shim binary, 6 types Phase 1 | 1 dev / 3 weeks |
| waza → MSBench value | Trajectory explorer, decoupled graders, dev lifecycle | Available today |
| MSBench → waza value | Scale compute, containers, Kusto, telemetry | Via import bridge |
| Architecture | Option C (one-way export) → evolve to B | 1 dev / 4-5 weeks initial |

**Total Phase 1 investment:** 1 dev / 5 weeks for export + grader shim.
**Total Phase 1+2 investment:** 1-2 dev / 9 weeks for full round-trip.

The bet: skill authors get fast local iteration AND statistical rigor at scale, without switching tools or learning MSBench's internals. waza is the steering wheel. MSBench is the engine.

---

## 12. Real MSBench Schemas (from Azure CLI benchmark)

With the exact MSBench benchmark format from the Azure CLI benchmark now confirmed, we have ground truth for the export target. This section documents the actual data formats that `waza export --target msbench` must produce.

### A. benchmark_metadata.json — Confirmed Schema

**Schema:** JSON array of objects, 3 fields per instance.

```json
[
  {
    "instance_id": "list_subscription",
    "problem_statement": "List my azure subscription start with GithubCopilot using your tools from Azure MCP server",
    "eval_script": "parse_list_subscription.py"
  },
  {
    "instance_id": "function_deployment_skill",
    "problem_statement": "Create a simple HTTP-triggered function app in javascript that returns a random compliment from a predefined list in a JSON response. Then deploy it to azure under my subscription GithubCopilotForAzure-Testing. Use recommended defaults for deployment unless necessary to change. Deploy to eastus region. If you are able to deploy the app successfully respond with Mission Accomplished!!!",
    "eval_script": "parse_deployment_skill.py"
  },
  {
    "instance_id": "swa_deployment_skill",
    "problem_statement": "Run npm install to install dependencies. Then deploy this app to azure as Azure Static Web App under my subscription GithubCopilotForAzure-Testing. Use recommended defaults for deployment unless necessary to change. Deploy to eastus2 region. If you are able to deploy the app successfully respond with Mission Accomplished!!!",
    "eval_script": "parse_deployment_skill.py"
  },
  {
    "instance_id": "aca_deployment_skill",
    "problem_statement": "Create a simple containerized Node.js hello world app and deploy to Azure Container Apps using my subscription GithubCopilotForAzure-Testing in eastus2 region. Use recommended defaults for deployment unless necessary to change. If you are able to deploy the app successfully respond with Mission Accomplished!!!",
    "eval_script": "parse_deployment_skill.py"
  },
  {
    "instance_id": "app_service_deployment_skill",
    "problem_statement": "Create a discussion board application and deploy to Azure App Service using my subscription GithubCopilotForAzure-Testing in eastus2 region. Use recommended defaults for deployment unless necessary to change. If you are able to deploy the app successfully respond with Mission Accomplished!!!",
    "eval_script": "parse_deployment_skill.py"
  },
  {
    "instance_id": "cost_optimization_skill",
    "problem_statement": "Find orphaned and unused resources in my subscription GithubCopilotForAzure-Testing that I can delete.",
    "eval_script": "parse_cost_optimization_skill.py"
  },
  {
    "instance_id": "compliance_skill",
    "problem_statement": "Show me expired certificates and secrets in my Azure Key Vault under my subscription GithubCopilotForAzure-Testing",
    "eval_script": "parse_compliance_skill.py"
  },
  {
    "instance_id": "resource_visualization_skill",
    "problem_statement": "Generate a Mermaid diagram showing the architecture of my resource group AIEvaluationResources in my subscription GithubCopilotForAzure-Testing",
    "eval_script": "parse_resource_visualization_skill.py"
  },
  {
    "instance_id": "foundry_skill",
    "problem_statement": "Build a RAG application in Python with Microsoft Foundry using knowledge indexes. I need to set things up from scratch. Use recommended defaults for any configuration unless necessary to change.",
    "eval_script": "parse_foundry_skill.py"
  }
]
```

**Key observations:**

- **Simple 3-field schema** — No nesting, no optional fields. Just `instance_id`, `problem_statement`, `eval_script`.
- **`problem_statement` IS the task prompt** — This is a direct 1:1 mapping to waza's `tasks[].prompt` field. The entire user-facing task instruction goes here as a string.
- **`eval_script` is a filename reference** — Points to a Python file in `docker/eval/` (e.g. `parse_deployment_skill.py`). Not inline Python, not a path with directories—just the filename.
- **Multiple instances share eval scripts** — Of the 9 instances above, 5 share `parse_deployment_skill.py`. MSBench doesn't require unique eval scripts per task.
- **`instance_id` is kebab-case** — Follows lowercase + hyphens naming convention (e.g., `list_subscription`, `cost_optimization_skill`).
- **waza mapping:**
  ```
  waza eval.yaml → MSBench benchmark_metadata.json
  tasks[n].name → instance_id
  tasks[n].prompt (or file content) → problem_statement
  graders[] (compiled to Python) → eval_script
  ```

### B. metadata.csv — Confirmed Format

```
instance_id,problem_statement,image_tag,patch
list_subscription,List my azure subscription start with GithubCopilot using your tools from Azure MCP server,azure.eval.x86_64.list_subscription:msbench-0.0.1,
function_deployment_skill,Create a simple HTTP-triggered function app...,azure.eval.x86_64.function_deployment_skill:msbench-0.0.1,
```

**Schema:** 4 columns: `instance_id`, `problem_statement`, `image_tag`, `patch`.

**Column details:**

- **`instance_id`** — Matches benchmark_metadata.json instance_id. Kebab-case.
- **`problem_statement`** — Matches benchmark_metadata.json problem_statement. Repeated here for convenience/querying in CSV.
- **`image_tag`** — Docker image tag for this instance. Format: `{benchmark_name}.eval.x86_64.{instance_id}:{version}`. Example: `azure.eval.x86_64.list_subscription:msbench-0.0.1`.
  - Benchmark name (e.g., `azure`) is likely derived from the benchmark directory or metadata file name.
  - Version (e.g., `msbench-0.0.1`) comes from a separate `version.txt` file.
  - This is generated by merging benchmark_metadata.json + version.txt, not manually written.
- **`patch`** — Empty for these instances. This column is used in SWE-bench style benchmarks where tasks have code diffs to apply. For benchmarks without patches, it's blank.

**Generation:**
```bash
# Conceptually:
# benchmark_metadata.json + version.txt → metadata.csv
# For each instance in benchmark_metadata.json:
#   image_tag = f"{benchmark_name}.eval.x86_64.{instance_id}:{version_string}"
#   patch = "" (empty)
```

### C. Agent/Model Configuration — Confirmed Format

**Source:** `{agent_name}_models.yaml` (e.g., `github-copilot-cli_models.yaml`)

```yaml
claude-sonnet-4.5-autodev-test:
  script: |
    export GITHUB_COPILOT_API_TOKEN=$(az account get-access-token --scope api://17b0ad65-ed36-4194-bb27-059c567bc41f/.default --query accessToken --output tsv)
    export GITHUB_COPILOT_INTEGRATION_ID=autodev-test
    export COPILOT_API_URL=https://ces-dev1.azurewebsites.net/api/copilot
    export COPILOT_AGENT_MODEL=sweagent-capi:claude-sonnet-4.5
  description: "Claude Sonnet 4.5 via autodev-test integration and CES proxy"

claude-opus-4.5-autodev-test:
  script: |
    export GITHUB_COPILOT_API_TOKEN=$(az account get-access-token --scope api://17b0ad65-ed36-4194-bb27-059c567bc41f/.default --query accessToken --output tsv)
    export GITHUB_COPILOT_INTEGRATION_ID=autodev-test
    export COPILOT_API_URL=https://ces-dev1.azurewebsites.net/api/copilot
    export COPILOT_AGENT_MODEL=sweagent-capi:claude-opus-4.5
  description: "Claude Opus 4.5 via autodev-test integration and CES proxy"

gpt-5.2-codex-autodev-test:
  script: |
    export GITHUB_COPILOT_API_TOKEN=$(az account get-access-token --scope api://17b0ad65-ed36-4194-bb27-059c567bc41f/.default --query accessToken --output tsv)
    export GITHUB_COPILOT_INTEGRATION_ID=autodev-test
    export COPILOT_API_URL=https://ces-dev1.azurewebsites.net/api/copilot
    export COPILOT_AGENT_MODEL=sweagent-capi:gpt-5.2-codex
  description: "GPT-5.2 Codex via autodev-test integration and CES proxy"
```

**Key insights:**

- **YAML key format:** `{model_name}-{integration_alias}` (e.g., `claude-sonnet-4.5-autodev-test`).
- **`script` field:** Shell script snippet that exports environment variables. This is sourced before the agent runs.
- **`description` field:** Human-readable text describing the model and integration.
- **Token acquisition:** `az account get-access-token --scope api://17b0ad65-ed36-4194-bb27-059c567bc41f/.default --query accessToken --output tsv`. This is Microsoft Entra ID (AAD) integration, not a static API key.
- **Integration ID:** `autodev-test` — This is a registered integration with the CES (Code Execution Service) proxy.
- **API URL:** `https://ces-dev1.azurewebsites.net/api/copilot` — The CES proxy endpoint in dev environment.
- **Model format:** `sweagent-capi:{model_name}` — The actual model identifier passed to the proxy (e.g., `sweagent-capi:claude-sonnet-4.5`).
- **Supported models:** claude-sonnet-4.5, claude-opus-4.5, gpt-5.2-codex, gpt-5.2, gemini-2.5-pro (can extend as needed).

**Mapping from waza perspective:**
- waza's `--model` flag (e.g., `--model claude-sonnet-4.5-autodev-test`) maps to a YAML key in the models file.
- The corresponding `script` value is sourced, setting `COPILOT_AGENT_MODEL` to the proxy-compatible model ID.
- This decouples waza's model naming from MSBench's agent invocation internals.

### D. Dockerfile — Confirmed Pattern

```dockerfile
FROM debian:bookworm-slim
ARG INSTANCE_ID
RUN mkdir -p /tmp/install
COPY vendor /tmp/install/vendor
RUN bash /tmp/install/vendor/install_all.sh
RUN apt-get update && apt-get install -y patch libicu-dev git
RUN curl -fsSL https://aka.ms/install-azd.sh | bash
RUN azd config set auth.useAzCliAuth true
RUN mkdir /output /testbed
COPY packages/${INSTANCE_ID}.tar.gz /tmp/install/drop.tar.gz
RUN tar -C / -xzf /tmp/install/drop.tar.gz
RUN rm -r /tmp/install
RUN chmod +x /entry.sh
CMD ["/entry.sh"]
```

**Key observations:**

- **Base image:** `debian:bookworm-slim` — Standard Debian, not Alpine, not a product-specific image. This is the foundation for all instances.
- **`ARG INSTANCE_ID`** — Build-time argument passed during `docker build --build-arg INSTANCE_ID=list_subscription`. This is how each instance gets a unique image.
- **Vendor bootstrap:** `vendor/install_all.sh` — Installed once, includes Python, Node, common system dependencies. This is shared across all instances and provided by the MSBench team.
- **Product-specific dependencies:** `curl -fsSL https://aka.ms/install-azd.sh | bash` and `azd config set auth.useAzCliAuth true` — These are specific to the Azure benchmark. Other benchmarks may add different tools (git, terraform, etc.).
- **Required directories:** `/output` for results, `/testbed` for the task workspace.
- **Per-instance package:** `packages/${INSTANCE_ID}.tar.gz` — A tar.gz archive created by MSBench's `prepare_metadata.py`. Extracted to `/` (root) of the container.
- **Entry point:** `/entry.sh` — Must be present in the extracted tar.gz. This is the bootstrap script for the task.
- **File permissions:** `RUN chmod +x /entry.sh` — Entry script must be executable.

**Container runtime environment at eval-script execution:**
- Working directory: `/testbed` (default or set by `/entry.sh`)
- Output directory: `/output/` (must exist for results)
- Agent: Injected as environment variables (from `{agent}_models.yaml` script)
- Eval script: In `docker/eval/` directory (location TBD in tar.gz extraction)

### E. Per-Instance Package Contents (Inferred)

MSBench's `prepare_metadata.py` creates `packages/{instance_id}.tar.gz`. When extracted to `/` in the container:

```
/entry.sh                          # Entrypoint script
/docker/eval/parse_*.py            # Eval scripts
/testbed/                          # (or populated by entry.sh)
  └── (task fixture files)
```

**Expected behavior:**
1. Dockerfile extracts tar.gz to `/`, making `/entry.sh` available.
2. Container CMD runs `/entry.sh`.
3. `/entry.sh` sets up `/testbed` with fixture files, sources the agent environment script, and invokes the eval script.
4. Eval script writes results to `/output/`.

### F. Updated waza → MSBench Export Mapping

With confirmed schemas, the field mapping is now precise:

| waza eval.yaml | MSBench benchmark_metadata.json | MSBench metadata.csv | Notes |
|---|---|---|---|
| `name` | (used as benchmark_name prefix) | benchmark_name in image_tag | e.g., `azure` from `azure-cli-benchmark` |
| `tasks[].name` | `instance_id` | `instance_id` | Kebab-case, unique per task |
| `tasks[].prompt` (or content of referenced file) | `problem_statement` | `problem_statement` | Full task instruction as string |
| `graders[]` (compiled to Python) | `eval_script` | (blank) | Filename only, e.g., `parse_deployment_skill.py` |
| `tasks[].context_files[]` | (bundled in tar.gz) | (blank) | Included in per-instance package |
| `config.model` | (entry in {agent}_models.yaml) | (blank) | Maps to YAML key, determines agent setup |
| `config.timeout_seconds` | (passed to container runtime) | (blank) | Docker timeout, environment variable, or orchestrator config |
| (version from waza or default) | (version.txt) | version in image_tag | e.g., `:msbench-0.0.1` |

**Export is a lossy downward projection:**
waza's `eval.yaml` is more expressive than MSBench needs (e.g., waza has `graders.config`, advanced grader types, environment setup). The export process filters to only what MSBench can consume: instance ID, prompt, and eval script filename. Everything else (advanced graders, program invocation, skill-specific logic) is either:
- Compiled into the Python eval script (if it's a Phase 1 grader type)
- Skipped with a warning (if it's a Phase 2 type like `behavior` or `prompt`)
- Bundled in the tar.gz (if it's a fixture file or helper script)

### G. What This Confirms for `waza export --target msbench` MVP

The format is straightforward enough that a minimal export is highly achievable:

1. **Read eval.yaml** → extract task names, prompts, graders, and fixture files.
2. **Generate `benchmark_metadata.json`** → iterate over tasks, map to 3-field schema.
3. **Compile Phase 1 graders to Python** → `code`, `regex`, `keyword`, `file`, `diff` → eval script files.
4. **Bundle context files** → create per-instance tar.gz with fixtures, eval scripts, and `/entry.sh` template.
5. **Generate `metadata.csv`** → derive from benchmark_metadata.json + version.txt.
6. **Copy Dockerfile template** → with INSTANCE_ID build arg.
7. **Generate `{agent}_models.yaml`** → from waza's `config.model` and known agent integrations.

**Blocking unknowns removed:**
- ✅ benchmark_metadata.json schema → confirmed (3 fields, array of objects)
- ✅ metadata.csv columns → confirmed (4 columns, image_tag pattern)
- ✅ Dockerfile pattern → confirmed (Debian base, vendor/ bootstrap, per-instance tar.gz)
- ✅ Agent config format → confirmed (YAML with script snippets)
- ❓ entry.sh template → inferred from Dockerfile (must source agent script, set /testbed, invoke eval script, write results to /output/)
- ❓ vendor/install_all.sh details → can be symlinked from MSBench repo or provided separately
- ❓ eval script output format → needs confirmation (JSON? stdout? exit codes?)
- ❓ Exact tar.gz directory layout → needs a sample, but pattern is clear

**MVP confidence level:** 🟢 **HIGH** — All critical schema pieces are confirmed. We can generate valid benchmark_metadata.json, metadata.csv, and Dockerfile. Remaining unknowns are implementation details that can be resolved through iteration with the MSBench team.

