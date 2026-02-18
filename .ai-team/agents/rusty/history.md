# History — Rusty

## Project Context
- **Project:** waza — CLI tool for evaluating Agent Skills
- **Stack:** Go (primary), React 19 + Tailwind CSS v4 (web UI)
- **User:** Shayne Boyer (spboyer)
- **Repo:** spboyer/waza
- **Universe:** The Usual Suspects

## Key Learnings

### Architecture
- **Model selection directive (2026-02-18):** Coding in Claude Opus 4.6, reviews in GPT-5.3-Codex, design in Gemini Pro 3
- **Web UI styling:** Keep clean and functional — colors close to DevEx dashboard, no fancy gradients
- **Agent execution:** Go engine drives CLI, web UI for visualization

### Code Quality
- Test coverage is non-negotiable
- Interface-based design for flexibility (AgentEngine, Validator patterns in Go)
- Functional options for configuration (Go convention)

### Team Structure
- Linus owns Go backend implementation
- Basher owns all testing strategy
- Livingston/Saul own documentation
- Richard Park available for Copilot SDK questions
