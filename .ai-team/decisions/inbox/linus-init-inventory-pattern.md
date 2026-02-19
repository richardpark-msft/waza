### 2026-02-20: waza init inventory pattern uses workspace + scaffold packages
**By:** Linus
**What:** `waza init` now uses `workspace.DetectContext` + `workspace.FindEval` for skill inventory, and `scaffold.EvalYAML()` / `scaffold.TaskFiles()` / `scaffold.Fixture()` for eval scaffolding. No separate eval generation path — reuses the same templates as `waza new`.
**Why:** Keeps one source of truth for eval templates (scaffold package). Any changes to `scaffold.EvalYAML()` template automatically propagate to both `waza new` and `waza init` scaffolding. The `scaffoldEvalSupportFiles` helper in `cmd_init.go` handles task/fixture creation separately from the summary display.
