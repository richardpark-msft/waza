# Skills Eval Framework

> Evaluate Agent Skills like you evaluate AI Agents

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Python](https://img.shields.io/badge/python-3.11+-blue.svg)](https://python.org)

A framework for evaluating [Agent Skills](https://agentskills.io/specification) using the same patterns and metrics that power AI agent evaluations. Measure whether your skills accomplish their intended goals with structured, reproducible tests.

## Features

- 🎯 **Task Completion Metrics** - Did the skill accomplish the goal?
- 🔍 **Trigger Accuracy Testing** - Is the skill invoked on the right prompts?
- 📊 **Behavior Quality Analysis** - Tool calls, efficiency, reasoning patterns
- 🤖 **Multiple Grader Types** - Code-based, LLM-as-judge, human review
- 📈 **JSON Reports** - Machine-readable results aligned with agent eval standards
- 🔄 **CI/CD Ready** - Run in GitHub Actions or any CI pipeline
- 🧩 **Eval-as-Skill** - Meta-evaluation capability within skill runtimes
- 🔬 **Real Integration Testing** - Use Copilot SDK for actual LLM responses
- 📊 **Model Comparison** - Compare results across different models
- 📡 **Runtime Telemetry** - Capture and analyze production metrics

## Quick Start

```bash
# Install
pip install skill-eval

# Install with Copilot SDK for real integration tests
pip install skill-eval[copilot]

# Initialize eval suite for a skill
skill-eval init my-skill

# Run evals (mock executor - fast, no API calls)
skill-eval run my-skill/eval.yaml

# Run with specific model
skill-eval run my-skill/eval.yaml --model claude-sonnet-4-20250514

# Run with real Copilot SDK (requires authentication)
skill-eval run my-skill/eval.yaml --executor copilot-sdk

# Compare results across models
skill-eval compare results-gpt4o.json results-claude.json
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `skill-eval run` | Run an evaluation suite |
| `skill-eval init` | Scaffold a new eval suite |
| `skill-eval compare` | Compare results across runs/models |
| `skill-eval analyze` | Analyze runtime telemetry |
| `skill-eval report` | Generate reports from results |
| `skill-eval list-graders` | List available grader types |

## Concepts

This framework aligns with established agent evaluation patterns:

| Concept | Description |
|---------|-------------|
| **Task** | A single test case with inputs and success criteria |
| **Trial** | One execution attempt of a task (multiple trials for consistency) |
| **Grader** | Logic that scores an aspect of skill performance |
| **Transcript** | Full record of skill execution (tool calls, outputs) |
| **Outcome** | Final state after skill execution |
| **Eval Suite** | Collection of tasks for a specific skill |

## Eval Specification

Define evals in YAML:

```yaml
# eval.yaml
name: my-skill-eval
skill: my-skill
version: "1.0"

config:
  trials_per_task: 3
  timeout_seconds: 300
  executor: mock                    # or copilot-sdk for real tests
  model: claude-sonnet-4-20250514   # model for execution

metrics:
  - name: task_completion
    weight: 0.4
    threshold: 0.8
  - name: trigger_accuracy
    weight: 0.3
    threshold: 0.9
  - name: behavior_quality
    weight: 0.3
    threshold: 0.7

graders:
  - type: code
    name: output_validation
    script: graders/validate.py
  - type: llm
    name: quality_check
    rubric: graders/quality_rubric.md

tasks:
  - include: tasks/*.yaml
```

## Executor Types

| Executor | Use Case | Requires |
|----------|----------|----------|
| `mock` | Unit tests, CI/CD | Nothing |
| `copilot-sdk` | Integration tests, benchmarking | Copilot auth |

```bash
# Fast mock execution (default)
skill-eval run eval.yaml

# Real Copilot SDK execution
skill-eval run eval.yaml --executor copilot-sdk --model gpt-4o
```

## Model Comparison

Compare skill performance across different models:

```bash
# Run with different models
skill-eval run eval.yaml --model gpt-4o -o results-gpt4o.json
skill-eval run eval.yaml --model claude-sonnet-4-20250514 -o results-claude.json
skill-eval run eval.yaml --model gpt-4o-mini -o results-mini.json

# Generate comparison report
skill-eval compare results-*.json -o comparison.md
```

Output:
```
              Summary Comparison              
┏━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━━┳━━━━━━━━━━━━━┓
┃ Metric          ┃ gpt-4o ┃ claude  ┃ gpt-4o-mini ┃
┡━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━━━╇━━━━━━━━━━━━━┩
│ Pass Rate       │ 100.0% │  95.0%  │      85.0%  │
│ Composite Score │   0.98 │   0.92  │        0.81 │
└─────────────────┴────────┴─────────┴─────────────┘

🏆 Best: gpt-4o (score: 0.98)
```

## Task Definition

```yaml
# tasks/example-task.yaml
id: example-001
name: Example Task
description: Test a specific skill capability

inputs:
  prompt: "Do something specific"
  context:
    files: [example.py]

expected:
  outcomes:
    - type: task_completed
  tool_calls:
    required:
      - pattern: "some_tool"
    forbidden:
      - pattern: "dangerous_operation"
```

## Results Format

```json
{
  "eval_id": "my-skill-eval-20260131",
  "skill": "my-skill",
  "config": {
    "model": "claude-sonnet-4-20250514",
    "executor": "copilot-sdk",
    "trials_per_task": 3
  },
  "summary": {
    "total_tasks": 10,
    "passed": 8,
    "failed": 2,
    "pass_rate": 0.8,
    "composite_score": 0.82
  },
  "metrics": {
    "task_completion": { "score": 0.85, "passed": true },
    "trigger_accuracy": { "score": 0.95, "passed": true }
  }
}
```

## Runtime Telemetry

Capture metrics from skills running in production:

```bash
# Analyze telemetry files
skill-eval analyze telemetry/sessions.json

# Filter to specific skill
skill-eval analyze telemetry/ --skill azure-deploy -o analysis.json
```

See [Telemetry Guide](docs/TELEMETRY.md) for integration patterns.

## GitHub Actions

```yaml
- uses: your-org/skill-eval-action@v1
  with:
    eval-path: ./my-skill/eval.yaml
    fail-on-threshold: true
```

## Documentation

- [Tutorial: Writing Skill Evals](docs/TUTORIAL.md)
- [Grader Reference](docs/GRADERS.md)
- [Integration Testing with Copilot SDK](docs/INTEGRATION-TESTING.md)
- [Runtime Telemetry](docs/TELEMETRY.md)
- [Demo Script](DEMO-SCRIPT.md)

## References

- [Anthropic - Demystifying Evals for AI Agents](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)
- [Agent Skills Specification](https://agentskills.io/specification)
- [OpenAI Evals](https://github.com/openai/evals)

## License

MIT
