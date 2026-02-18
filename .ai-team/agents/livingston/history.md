# History — Livingston

## Project Context
- **Project:** waza — CLI tool for evaluating Agent Skills
- **Stack:** Go (primary), React 19 + Tailwind CSS v4 (web UI)
- **User:** Shayne Boyer (spboyer)
- **Repo:** spboyer/waza
- **Universe:** The Usual Suspects

## Key Learnings

### Documentation Structure
- **Main files:** README.md, docs/, waza-go/README.md
- **Key sections:** Usage, examples, CLI flags, agent integration
- **API docs:** BenchmarkSpec, TestCase, EvaluationOutcome, Validator interface
- **Update requirement:** Must stay in sync with code changes

### Waza Concepts
- Evaluation specs (YAML format)
- Task definitions with fixtures
- Validator registry (extensible grading)
- Agent execution (Go engine, fixture isolation)
- Results and scoring

### CI/CD
- Workflows defined in .github/workflows/
- Branch protection enforces docs stay current
- Changelog tracking for releases
