### 2026-02-17: Grader registry near-complete — only prompt grader remains
**By:** Linus (Backend Dev)
**Related:** #28, PR #179
**What:** Implemented keyword, json_schema, and program graders. The `Create()` factory now handles 10 of 11 grader kinds. Only `GraderKindPrompt` remains unimplemented. The json_schema grader uses `santhosh-tekuri/jsonschema/v6` which was already an indirect dependency. The program grader passes agent output via stdin and workspace dir via `WAZA_WORKSPACE_DIR` env var — any future graders that shell out should follow this convention.
**Why:** Completing the grader registry unblocks eval authors who need keyword matching, schema validation, or custom external grading scripts.
