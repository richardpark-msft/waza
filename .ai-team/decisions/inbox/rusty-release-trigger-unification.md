# Decision: Unified Release Trigger & Version Single Source-of-Truth

**Date:** Feb 20, 2026  
**Author:** Rusty (Architect)  
**Status:** PROPOSED  
**Impact:** Release process, artifact consistency, extension users

## Problem

The release process is fragmented across two independent workflows:
- `go-release.yml` → produces CLI binaries + GitHub Release on `v*` tags
- `azd-ext-release.yml` → produces extension binaries + registry update on version.txt/extension.yaml changes

This causes:
1. **Version desync**: extension.yaml (0.3.0) lags behind CLI version (0.4.0-alpha.1)
2. **registry.json stale**: doesn't reflect latest CLI binaries
3. **Dual tag schemes**: CLI uses `v*`, extension uses `azd-ext-microsoft-azd-waza_*`
4. **No validation**: easy for files to drift without detection

## Solution

### Canonical Release Trigger
- **Source of truth**: Git tag `v*.*.*` (semver)
- **Single coordinator**: `release.yml` workflow
- Both CLI and extension release from the same tag push

### Version File Synchronization
- `setup-version` job in release.yml extracts version from tag
- Pre-flight validation job: confirm version.txt == tag (fail if not)
- Sync-versions job runs **BEFORE builds** (not after) to update version.txt, extension.yaml
- Version is passed to all downstream build jobs via job outputs

### Retirement Plan
- `go-release.yml` — disable or remove once release.yml is stable (keep as reference)
- `azd-ext-release.yml` — disable or remove once release.yml is stable (keep as reference)

## Tradeoffs

| Approach | Pros | Cons |
|----------|------|------|
| Current (two workflows) | Decoupled; can release CLI/ext separately | Frequent desync; confusing; hard to audit |
| New (one workflow, tag-driven) | Single source-of-truth; hermetic; auditable | Requires tag == version.txt (manual step) |
| Alternative (version.txt-driven) | No manual tagging | Requires version.txt as source-of-truth; less Git-native |

**Chosen:** Tag-driven (New). Rationale: Git tags are immutable, auditable, and align with GitHub release workflow. Manual version bump + tag is a single commit, hard to desync.

## Implementation Notes

1. **Pre-flight validation** must happen before build jobs
2. **Version sync** must be idempotent (safe to re-run)
3. **registry.json** update must be in a separate PR (not blocking main release)
4. **Changelog** should be manually maintained (auto-generate breaks if commits are missing)

## Risks

- If someone pushes tag without updating version.txt, release will fail → teach developers the process
- Artifact URLs in registry.json are hardcoded to spboyer/waza → if repo moves, links break (acceptable for now, document as limitation)
- No way to release CLI-only or extension-only (both happen together) → acceptable, rare use case

## Decision

**APPROVED** — Implement release.yml as unified trigger, retire go-release.yml + azd-ext-release.yml once stable.

---

**See Also:**
- Issue #223 (Release Process Normalization)
- Triage comment on issue #223
- .ai-team/agents/rusty/history.md (audit findings)
