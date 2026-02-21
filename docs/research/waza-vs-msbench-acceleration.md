# Waza vs. MSBench Acceleration: Strategic Gap Analysis

**Date:** February 21, 2026
**Author:** Rusty (Lead/Architect), requested by Shayne Boyer
**Context:** MSBench is building a repeatable quality loop for GitHub Copilot products (CCA, CLI, VS Code Agent). This analysis maps their 12 platform areas against waza v0.8.0 capabilities.

---

## 1. Feature-by-Feature Comparison

MSBench self-assessed against Aether (OpenAI's eval UX) across 12 areas. Here's where waza stands relative to both.

| # | MSBench Area | MSBench vs Aether | Waza v0.8.0 Status | Waza vs MSBench | Notes |
|---|---|---|---|---|---|
| 1 | **Run Details** | Parity/Better (missing exact CLI shown) | ✅ Full run metadata, CLI flags captured in results JSON, model/config in output | **Parity** | Waza records executor, model, config, timing. MSBench notes they're missing "exact command line shown" — waza doesn't persist the literal CLI invocation either |
| 2 | **Task Results** | Parity/Better | ✅ Per-task pass/fail, scores, grader verdicts, group_by dimensions | **Parity** | Both solid here. Waza adds `group_by` for slicing results by model/dimension |
| 3 | **Task Details (Metrics)** | WORSE — metrics coupled to agent, no LLM summary | ✅ 11 grader types, decoupled from executor. `prompt` grader for LLM-as-judge. `--interpret` for plain-language summaries | **Waza AHEAD** | MSBench's biggest pain: metrics tightly coupled to the agent. Waza's graders are fully decoupled — any grader works with any executor. LLM explanations via `prompt` grader + `--interpret` flag |
| 4 | **Task Details (Trajectories)** | MISSING — WIP on ATIF, no rich explorer | ✅ Aspire-style waterfall timeline, event-by-event tool trace, expandable details, session digest (tokens/turns) | **Waza AHEAD** | This is waza's strongest advantage. Full trajectory explorer with waterfall visualization, something MSBench explicitly marks as MISSING |
| 5 | **Overall Scores** | WORSE — calculated in UI not pipeline, no LLM explanation | ✅ Scores computed in pipeline. Weighted grader scores (`weight` field). `--interpret` for LLM explanations. Bootstrap CI with significance badges | **Waza AHEAD** | Waza computes scores server-side in the Go pipeline, not in the UI layer. Grader weighting and statistical CI are pipeline-native |
| 6 | **Grader Analysis (Repeatability)** | MISSING — no repeatability measurement | ⚠️ Partial — `trials_per_task` runs N trials, bootstrap CI measures variance, but no dedicated grader-specific repeatability analysis | **Waza Partial** | Waza can run multiple trials and compute CI, but doesn't isolate *grader* variance from *agent* variance. MSBench wants to answer "is this grader reliable?" — waza answers "is this score stable?" |
| 7 | **Compare Runs** | Near Parity — missing custom metrics extensibility | ✅ `waza compare` CLI + CompareView dashboard with metric deltas, per-task score/outcome/duration comparison, trajectory diffing | **Waza AHEAD** | Waza's CompareView includes trajectory-level diffing between runs — something MSBench doesn't mention. Both lack custom metric extensibility |
| 8 | **Classify & Suggest** | MISSING — need broader than just prompt changes | ✅ `waza suggest` generates eval artifacts from SKILL.md. `waza dev --copilot` suggests frontmatter improvements. No automatic failure clustering/classification | **Split** | Waza suggests *skill improvements*, MSBench wants *failure classification* (root-cause clustering). Different problems. Waza has the suggest infrastructure but not the classify-failures piece |
| 9 | **Prompt Optimizer** | MISSING — automated hill-climb | ⚠️ `waza dev` iterative loop with `--auto` mode, but optimizes skill frontmatter compliance, not prompt performance. No hill-climb against eval scores | **MSBench AHEAD (in vision)** | Neither has this today. MSBench plans 2 dev/2 months. Waza's `dev` loop is iterative but targets compliance, not score optimization. True prompt hill-climbing is a gap |
| 10 | **Registry** | WORSE — evolving from static to dynamic | ⚠️ `registry.json` for azd extension distribution. Skills directory + template scaffolding. No shared asset registry for datasets/graders/configs | **MSBench AHEAD (in vision)** | MSBench plans a full asset registry (2 dev/3 months). Waza has no equivalent — graders, datasets, and configs aren't shareable/discoverable across projects |
| 11 | **Human Review** | MISSING | ❌ Not implemented. `human` grader doc exists as placeholder only | **Neither** | Both lack this. The doc in waza acknowledges the gap |
| 12 | **Reporting** | WORSE — need historical trends, pass@K, qualitative | ✅ TrendsPage dashboard (pass rate, tokens, cost, duration over time). Model filtering. No pass@K. JUnit XML for CI. CSV export | **Waza Partial** | Waza has trend charts and CI integration reporters. Missing: pass@K metric, qualitative/narrative reporting, historical cross-run aggregation beyond what's loaded in dashboard |

---

## 2. Where Waza is AHEAD

These are areas where MSBench marks themselves as MISSING or WORSE, and waza has shipped solutions.

### 🏆 Trajectory Explorer (MSBench: MISSING)

**MSBench pain:** "Have swebench format, WIP on ATIF, no rich trajectory explorer."
**Waza reality:** Full Aspire-style waterfall timeline since v0.8.0. Tool-call-by-tool-call trace with timing, expandable event details, session digest cards showing token/turn metrics. Toggle between Timeline and Events modes. This is production-ready, not WIP.

**Strategic value:** MSBench estimates 2 dev/2 months to build trajectory tracing. Waza already ships it. This is our single biggest competitive advantage.

### 🏆 Decoupled Grading Platform (MSBench: WORSE)

**MSBench pain:** "Metrics coupled to agent, no LLM summary/explanation, WIP on extensibility."
**Waza reality:** 11 grader types, all decoupled from executors via the Validator interface. Any grader works with any AgentEngine (mock, copilot-sdk). `prompt` grader provides LLM-as-judge with rubrics. `--interpret` flag gives plain-language explanations. Grader weighting is pipeline-native.

**Strategic value:** MSBench's #3 priority (1 dev/2 months). Waza's grader architecture is already what they're trying to build.

### 🏆 Multi-Run Comparison with Trajectory Diffing (MSBench: Near Parity)

**MSBench status:** "Missing custom metrics extensibility."
**Waza bonus:** CompareView includes `TaskTrajectoryCompare` — side-by-side trajectory diffing between runs. This goes beyond what MSBench describes. CLI `waza compare` also works headlessly for CI.

### 🏆 Overall Scores in Pipeline (MSBench: WORSE)

**MSBench pain:** "Calculated in UI not pipeline."
**Waza reality:** All scoring happens in Go pipeline. Weighted scores, bootstrap CI, significance testing — all computed server-side. Dashboard renders results, doesn't calculate them.

### 🏆 Skill Development Toolchain (MSBench: No equivalent)

MSBench is purely an evaluation platform. Waza provides a complete skill development lifecycle:
- `waza init` / `waza new` — scaffolding
- `waza suggest` — LLM-generated eval artifacts
- `waza dev` — iterative compliance improvement
- `waza check` — submission readiness
- `waza tokens` — token budget management
- Trigger testing — prompt accuracy metrics

MSBench has nothing in this space. They focus on measuring quality after the skill exists; waza helps *build* quality in.

---

## 3. Where MSBench is AHEAD (or Plans to Be)

### 🔴 Trajectory Auto-Analysis / Failure Clustering (Planned: 1 dev/2 months)

**What MSBench wants:** Automatic clustering of failure root causes. "Highlight key trace segments" — an LLM reads trajectories and says "these 15 failures all hit the same tool-call-timeout pattern."

**Waza gap:** `waza suggest` recommends improvements to skills, but doesn't analyze *failure patterns across runs*. No clustering, no automatic root-cause categorization. A human must inspect trajectories one-by-one in the dashboard.

**Priority for waza:** HIGH. This multiplies the value of our existing trajectory data. We have the traces — we need the analysis layer.

### 🔴 Grader Repeatability Analysis (Planned: 1 dev/2 months)

**What MSBench wants:** "Quantify score variance, prevent false wins/losses." Run the same grader on the same output N times and measure how stable the score is. Specifically targets LLM-as-judge drift.

**Waza gap:** `trials_per_task` measures *end-to-end* variance (agent + grader combined). No way to isolate grader variance independently. If a `prompt` grader gives 0.8, 0.6, 0.9 on the same output, waza can't tell you that.

**Priority for waza:** MEDIUM-HIGH. As LLM-as-judge becomes more central (our `prompt` grader), grader reliability becomes critical. Without this, we can't tell if a score change is real or judge noise.

### 🔴 Automated Prompt Optimization / Hill-Climbing (Planned: 2 dev/2 months)

**What MSBench wants:** Automated intervention loop. Run eval → identify weak area → generate prompt tweak → re-eval → keep if better → repeat. Target: 32 cycles/quarter.

**Waza gap:** `waza dev --auto` iterates on *frontmatter compliance*, not *eval score*. No mechanism to: (a) identify weakest tasks, (b) generate targeted prompt modifications, (c) re-run and compare, (d) accept/reject automatically.

**Priority for waza:** MEDIUM. This is ambitious (MSBench estimates 2 dev/2 months). Waza's `dev` loop is the right foundation but needs to close the loop with eval scores, not just compliance.

### 🔴 Asset Registry (Planned: 2 dev/3 months)

**What MSBench wants:** "Reuse datasets/graders/configs, self-serve." A shared catalog where teams can discover and compose eval components.

**Waza gap:** No shared registry. Each project is self-contained. Graders, tasks, and fixtures can't be published or discovered across projects. The `registry.json` is only for azd extension distribution.

**Priority for waza:** LOW-MEDIUM. Important for enterprise adoption but not for waza's current audience (individual skill developers). Would matter more if waza targets team-scale adoption.

### 🟡 Operating Model: Red Zones + Hill-Climb Cycles

MSBench's operating model (Ingest → Diagnose → Intervene → Validate → Ship & Lock-in, targeting 32 cycles/quarter) is more systematic than waza's approach. Waza provides the *tools* but not the *process orchestration*.

**Waza gap:** No concept of "red zones" (prioritized gaps), no cycle tracking, no lock-in mechanism (preventing regression on improved areas).

**Priority for waza:** LOW for tooling, HIGH for documentation. Waza could document a recommended operating model without building new features.

---

## 4. Strategic Gaps — What Waza Should Build Next

Ranked by impact and alignment with MSBench's validated priorities.

### Tier 1: Build Now (High Impact, Leverage Existing Strengths)

| Gap | Effort | Why Now |
|---|---|---|
| **Trajectory auto-analysis** | 1 dev / 2–3 weeks | We already capture full trajectories. Adding LLM-powered clustering and root-cause extraction turns our best feature into a 10x feature. Use our own `prompt` grader infrastructure to analyze traces |
| **Grader repeatability mode** | 1 dev / 1–2 weeks | Add `--repeatability` flag that re-grades the same output N times and reports grader variance. Small change to the orchestration layer. Critical for `prompt` grader trust |
| **pass@K metric** | 0.5 dev / 1 week | We already run multiple trials. Computing pass@K (probability of at least one correct answer in K attempts) is a straightforward addition to scoring |

### Tier 2: Build Soon (Medium Impact, Competitive Parity)

| Gap | Effort | Why Soon |
|---|---|---|
| **Failure classification** | 1 dev / 3–4 weeks | Post-run analysis that clusters failures by pattern (timeout, wrong tool, hallucination, etc). Dashboard view for failure categories |
| **Score-based hill-climbing** | 1 dev / 4–6 weeks | Extend `waza dev` to optimize against eval scores, not just compliance. Run → identify weakest task → suggest prompt change → re-run → accept if score improves |
| **Historical trend persistence** | 1 dev / 2 weeks | Store results across runs in a local SQLite or append-only JSON log. Currently trends only show what's loaded in the dashboard session |

### Tier 3: Build Later (Strategic, But Lower Urgency)

| Gap | Effort | Why Later |
|---|---|---|
| **Asset registry** | 2 dev / 6–8 weeks | Shared grader/dataset/config catalog. Only matters at team scale. MSBench's estimate of 2 dev/3 months seems right |
| **Human review workflow** | 1 dev / 4 weeks | Dashboard UI for human-in-the-loop grading. Neither waza nor MSBench has this today |
| **Custom metric extensibility** | 1 dev / 2 weeks | Plugin interface for user-defined metrics beyond built-in graders. Both waza and MSBench lack this |

---

## 5. Opportunities — Where Waza Could Fill MSBench's Gaps

These are integration points where waza could provide value to MSBench teams directly.

### 🎯 Opportunity 1: Trajectory Explorer as MSBench Component

MSBench marks trajectory visualization as MISSING (2 dev/2 months investment planned). Waza's waterfall timeline is production-ready.

**Integration path:** Waza's dashboard could ingest MSBench result format (or MSBench could export to waza-compatible JSON). The React components (`WaterfallTimeline`, `TrajectoryViewer`, `TaskTrajectoryCompare`) are embeddable.

**Effort to adapt:** 1 dev / 2–3 weeks for format bridging.

### 🎯 Opportunity 2: Grader Library for MSBench

MSBench wants to "decouple graders from agents" (1 dev/2 months). Waza's 11-grader Validator registry is already decoupled.

**Integration path:** Waza's grader types (especially `program`, `inline_script`, `prompt`) could be exposed as a grader SDK that MSBench calls. The `ValidatorRegistry` pattern is designed for extensibility.

**Effort to adapt:** 2 dev / 3–4 weeks for SDK packaging.

### 🎯 Opportunity 3: Skill Development Pipeline Feeding MSBench

Waza's strength is the development loop (init → new → suggest → dev → check → run). MSBench's strength is the validation loop (ingest → diagnose → intervene → validate → ship).

**Integration path:** `waza run` produces results → MSBench ingests for cross-product comparison. Waza handles skill authoring quality, MSBench handles product-level quality signals.

**This is the complementary-tools pattern** we already identified with skill-validator, now at a larger scale.

### 🎯 Opportunity 4: Operating Model Documentation

MSBench's "32 hill-climb cycles/quarter" operating model could be documented as a waza workflow guide. Map their Ingest → Diagnose → Intervene → Validate → Ship loop to waza commands:

| MSBench Phase | Waza Command |
|---|---|
| Ingest | `waza run` (collect baseline) |
| Diagnose | `waza serve` (trajectory inspection) + future auto-analysis |
| Intervene | `waza suggest` / `waza dev` |
| Validate | `waza run` (re-measure) + `waza compare` (delta) |
| Ship & Lock-in | `waza check` + CI regression suite |

---

## Summary Scorecard

| Dimension | Waza | MSBench | Verdict |
|---|---|---|---|
| Trajectory visualization | ✅ Shipped, rich | ❌ Missing, planned | **Waza wins** |
| Trajectory auto-analysis | ❌ Not built | ⏳ Planned | **MSBench wins (future)** |
| Grading platform | ✅ 11 types, decoupled | ⚠️ Coupled, evolving | **Waza wins** |
| Grader repeatability | ⚠️ Trial variance only | ⏳ Planned | **MSBench wins (future)** |
| Multi-run comparison | ✅ CLI + dashboard + trajectory diff | ✅ Near parity | **Waza slight edge** |
| Prompt optimization | ⚠️ Compliance only | ⏳ Planned | **MSBench wins (future)** |
| Asset registry | ❌ None | ⏳ Planned | **MSBench wins (future)** |
| Human review | ❌ Not built | ❌ Not built | **Tie (gap for both)** |
| Reporting / trends | ✅ Dashboard trends, JUnit | ⚠️ Worse, needs work | **Waza slight edge** |
| Skill dev toolchain | ✅ Full lifecycle | ❌ Not in scope | **Waza wins** |
| Statistical rigor | ✅ Bootstrap CI, significance | ⚠️ Calculated in UI | **Waza wins** |
| Operating model / process | ⚠️ Tools without process | ✅ Defined loop, cadence | **MSBench wins** |
| Scale (multi-product) | ⚠️ Single-skill focus | ✅ Cross-product platform | **MSBench wins** |

**Bottom line:** Waza is ahead on *shipped capabilities* — trajectory explorer, grading platform, comparison tooling, and developer experience. MSBench is ahead on *vision and process* — failure clustering, grader repeatability, automated optimization, and a systematic operating model. The highest-impact move for waza is adding analysis intelligence on top of the data we already capture (trajectory auto-analysis, failure clustering, grader repeatability). These are 4–6 week investments that would close the biggest gaps.
