# Writing Skill Evals - Tutorial

This tutorial walks you through creating evaluations for your Agent Skills.

## Prerequisites

- `waza` CLI installed:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/spboyer/waza/main/install.sh | bash
  ```
- An existing skill to evaluate

## Step 1: Initialize Your Eval Suite

You have several options to create your eval suite:

### Option A: Create a New Skill

The fastest way to get started is to create a new skill:

```bash
# Create a new skill (this scaffolds eval suite)
waza new my-awesome-skill

# The generator creates:
# my-awesome-skill/
# ├── SKILL.md                     # Skill definition
# ├── evals/
# │   ├── eval.yaml           # Main eval configuration
# │   ├── tasks/              # Generated task definitions
# │   │   ├── task-001.yaml
# │   │   ├── task-002.yaml
# │   │   └── task-003.yaml
# │   └── fixtures/           # Sample project files for context
# │       ├── sample.py
# │       └── sample.txt
```

<!-- Future feature: LLM-assisted generation with --assist flag -->
<!-- Future feature: GitHub repo discovery with --repo and --skill flags -->
<!-- Future feature: Scanning with --scan flag -->

## Step 2: Configure Your Eval Specification

Edit `evals/eval.yaml` to define your evaluation:

```yaml
name: my-awesome-waza
description: Evaluate the my-awesome-skill skill
skill: my-awesome-skill
version: "1.0"

config:
  trials_per_task: 3      # Run each task 3 times for consistency
  timeout_seconds: 300    # 5 minute timeout per trial
  parallel: false         # Run tasks sequentially

graders:
  - type: code
    name: basic_validation
    config:
      assertions:
        - "len(output) > 0"
        - "'error' not in output.lower()"

  - type: regex
    name: pattern_check
    config:
      must_match:
        - "(?i)(success|deployed)"
      must_not_match:
        - "(?i)(error|failed)"

tasks:
  - "tasks/*.yaml"
```

## Step 3: Write Task Definitions

Tasks are individual test cases. Create them in `evals/tasks/`:

```yaml
# evals/tasks/deploy-app.yaml
id: deploy-app-001
name: Deploy Simple App
description: Test deploying a basic application

inputs:
  prompt: "Deploy this app to Azure"

expected:
  output_contains:
    - "deploy"
    - "success"
```

## Step 4: Choose Your Graders

Waza supports multiple grader types for evaluation:

### Code Grader (Deterministic)
```yaml
- type: code
  name: output_check
  config:
    assertions:
      - "len(output) > 0"
      - "'success' in output.lower()"
```

### Regex Grader (Pattern Matching)
```yaml
- type: regex
  name: format_check
  config:
    must_match:
      - "deployed.*successfully"
    must_not_match:
      - "error|failed|exception"
```

<!-- Future feature: LLM grader for AI-based evaluation -->
<!-- Future feature: Script grader for custom validation logic -->

## Step 5: Run Your Evals

```bash
# Run all tasks
waza run evals/eval.yaml --context-dir evals/fixtures

# Run with verbose output
waza run evals/eval.yaml --context-dir evals/fixtures -v

# Save results to file
waza run evals/eval.yaml --context-dir evals/fixtures -o results.json

# Run specific task
waza run evals/eval.yaml --task deploy-app-001

# Run in parallel
waza run evals/eval.yaml --parallel --workers 4
```

### Progress Output

By default, the CLI shows progress:

```
Running evaluations for my-awesome-waza...
  ✓ deploy-app-001 passed
  ✓ advanced-task-001 passed

Results: 2/2 tasks passed ✓
```

Use `-v/--verbose` for detailed output with conversation snippets.

## Step 6: Interpret Results

### Console Output
```
Results: 2/2 tasks passed ✓
```

### JSON Output Structure
```json
{
  "eval_id": "my-awesome-waza-20260131-001",
  "skill": "my-awesome-skill",
  "summary": {
    "total_tasks": 2,
    "passed": 2,
    "failed": 0,
    "pass_rate": 1.0
  },
  "tasks": [...]
}
```

## Step 7: Next Steps

After running your evaluations:

```bash
# Check skill readiness
waza check

# View results in dashboard
waza serve
```

## Best Practices

1. **Start Simple**: Begin with basic code graders
2. **Multiple Trials**: Use 3+ trials for consistent results
3. **Clear Triggers**: Define explicit trigger phrases in your skill description
4. **Incremental Testing**: Add tasks as you find edge cases
5. **Track Baselines**: Store results to detect regressions

## Troubleshooting

### "No tasks found"
- Check your `tasks` glob pattern in eval.yaml
- Ensure task files have `.yaml` extension

### "Grader failed"
- Check assertion syntax (Python expressions)
- Verify context variables are available

## Next Steps

- Read the [Grader Reference](GRADERS.md)
- See [Example Evals](../examples/)
- Check out [Getting Started Guide](GETTING-STARTED.md)
