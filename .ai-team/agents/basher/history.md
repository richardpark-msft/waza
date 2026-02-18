# History — Basher

## Project Context
- **Project:** waza — CLI tool for evaluating Agent Skills
- **Stack:** Go (primary), React 19 + Tailwind CSS v4 (web UI)
- **User:** Shayne Boyer (spboyer)
- **Repo:** spboyer/waza
- **Universe:** The Usual Suspects

## Key Learnings

### Testing Strategy
- **Model directive:** Coding in Claude Opus 4.6 (same as production code)
- **Test types:** Go unit tests (*_test.go), integration tests, Playwright E2E
- **Fixture isolation:** Original fixtures never modified — tests work in temp workspace
- **Coverage goal:** Non-negotiable (Rusty's requirement)

### Waza-specific Tests
- TestCase execution scenarios
- BenchmarkSpec validation
- Validator registry functionality
- CLI flag handling
- Agent execution mocking

### CI/CD
- Branch protection requires tests to pass
- Go CI workflow in .github/workflows/go-ci.yml
- Test results tracked for quality assurance
