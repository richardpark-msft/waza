# Code Explainer Eval

Example eval suite for a hypothetical "code-explainer" skill that explains code snippets to users.

## Structure

```
code-explainer/
├── eval.yaml                    # Main eval configuration
├── tasks/                       # Individual test tasks
│   ├── explain-python-recursion.yaml
│   ├── explain-js-async.yaml
│   ├── explain-list-comprehension.yaml
│   └── explain-sql-join.yaml
├── graders/
│   └── explanation_quality.py   # Custom grader for explanation quality
└── trigger_tests.yaml           # Trigger accuracy tests
```

## Running

```bash
# Quick test with mock executor
skill-eval run examples/code-explainer/eval.yaml --executor mock -v

# Full test with Copilot SDK
skill-eval run examples/code-explainer/eval.yaml --executor copilot-sdk -v
```

## What It Tests

This eval tests a code explanation skill across:

| Dimension | Coverage |
|-----------|----------|
| **Languages** | Python, JavaScript, SQL |
| **Concepts** | Recursion, async/await, list comprehensions, JOINs |
| **Complexity** | Beginner to intermediate |

## Metrics

| Metric | Weight | Threshold | What It Measures |
|--------|--------|-----------|------------------|
| `task_completion` | 40% | 80% | Did the skill complete the explanation? |
| `trigger_accuracy` | 30% | 90% | Does it trigger on appropriate prompts? |
| `behavior_quality` | 30% | 70% | Tool usage, response time within limits? |

## Graders

### Global Graders (in eval.yaml)
- **has_explanation**: Output length > 10 chars
- **no_errors**: No fatal error patterns in output

### Custom Grader (explanation_quality.py)
Evaluates explanations on 5 criteria (20 points each):
1. Sufficient length (≥200 chars)
2. Structured sections (overview, steps, key points)
3. Language identification
4. Educational tone
5. No error indicators

Pass threshold: 60%

## Customizing

### Add a New Task

Create `tasks/explain-new-concept.yaml`:

```yaml
id: explain-new-concept-001
name: Explain New Concept
description: Test explaining X concept

tags:
  - language
  - concept

inputs:
  prompt: |
    Explain this code:
    ```python
    # Your code here
    ```
  context:
    language: python
    complexity: beginner
    concept: your-concept

expected:
  output_contains:
    - "keyword1"
  outcomes:
    - type: task_completed

graders:
  - name: explains_concept
    type: code
    config:
      assertions:
        - "len(output) > 10"
```

### Modify Trigger Tests

Edit `trigger_tests.yaml` to add prompts that should or shouldn't trigger the skill.
