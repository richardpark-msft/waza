#!/usr/bin/env python3
"""Custom grader for code-explainer skill.

This demonstrates how to create a custom grader that evaluates
the quality of code explanations based on multiple criteria.

Usage:
  Can be used as a custom grader in eval.yaml:
  
  graders:
    - type: custom
      name: explanation_quality
      config:
        script: graders/explanation_quality.py
"""

import json
import sys
import re
from typing import Any


def grade(context: dict[str, Any]) -> dict[str, Any]:
    """Grade a code explanation.
    
    Evaluates explanations on 5 criteria (20 points each):
    1. Sufficient length - meaningful explanation, not too brief
    2. Structured sections - overview, steps, key concepts
    3. Language identification - mentions the programming language
    4. Educational tone - explanatory language, not just code
    5. No error indicators - successfully completed the task
    
    Args:
        context: Dict containing:
            - output: The skill's response text
            - task: The task definition with inputs and expected outputs
            - skill_invoked: Whether the skill was triggered
            - tool_calls: List of tool calls made
            
    Returns:
        Dict with:
            - score: Float 0.0-1.0
            - passed: Boolean
            - message: Human-readable summary
            - details: Additional grading information
    """
    output = context.get("output", "")
    task = context.get("task", {})
    
    score = 0.0
    checks = []
    
    # Check 1: Minimum length (20 points)
    # A good explanation should be at least 200 characters
    min_length = 200
    if len(output) >= min_length:
        score += 0.2
        checks.append(f"✓ Sufficient length ({len(output)} chars >= {min_length})")
    else:
        checks.append(f"✗ Too short ({len(output)} chars < {min_length})")
    
    # Check 2: Has structured sections (20 points)
    # Look for organizational patterns in the explanation
    section_patterns = [
        (r"(?i)(overview|summary|introduction)", "overview"),
        (r"(?i)(step[\s-]?by[\s-]?step|step \d|first,|then,|finally,|1\.)", "steps"),
        (r"(?i)(key concept|important|note|remember)", "key points"),
    ]
    sections_found = []
    for pattern, name in section_patterns:
        if re.search(pattern, output):
            sections_found.append(name)
    
    if len(sections_found) >= 2:
        score += 0.2
        checks.append(f"✓ Has structured sections: {', '.join(sections_found)}")
    else:
        checks.append(f"✗ Missing structure (found: {sections_found or 'none'})")
    
    # Check 3: Identifies programming language (20 points)
    language = task.get("inputs", {}).get("context", {}).get("language", "")
    language_indicators = {
        "python": [r"(?i)\bpython\b", r"\bdef\b", r"\bimport\b", r"__\w+__"],
        "javascript": [r"(?i)\bjavascript\b", r"(?i)\bjs\b", r"\bfunction\b", r"\bconst\b", r"\blet\b"],
        "sql": [r"(?i)\bsql\b", r"(?i)\bquery\b", r"(?i)\bselect\b", r"(?i)\btable\b"],
        "java": [r"(?i)\bjava\b", r"\bclass\b", r"\bpublic\b", r"\bprivate\b"],
        "typescript": [r"(?i)\btypescript\b", r"(?i)\bts\b", r"\binterface\b", r"\btype\b"],
    }
    
    if language and language in language_indicators:
        patterns = language_indicators[language]
        matches = [p for p in patterns if re.search(p, output)]
        if matches:
            score += 0.2
            checks.append(f"✓ Identifies {language} ({len(matches)} indicators)")
        else:
            checks.append(f"✗ Does not clearly identify {language}")
    else:
        # No language specified or unknown language - give benefit of doubt
        score += 0.2
        checks.append("✓ Language check skipped (not specified)")
    
    # Check 4: Educational tone (20 points)
    # Look for explanatory language patterns
    educational_patterns = [
        r"(?i)this (code|function|method|query|snippet)",
        r"(?i)(returns|produces|creates|generates|outputs)",
        r"(?i)(means|indicates|represents|is used to)",
        r"(?i)(because|since|therefore|so that)",
        r"(?i)(for example|such as|like|e\.g\.)",
    ]
    edu_matches = [p for p in educational_patterns if re.search(p, output)]
    
    if len(edu_matches) >= 2:
        score += 0.2
        checks.append(f"✓ Educational tone ({len(edu_matches)} patterns)")
    else:
        checks.append(f"✗ Could be more educational ({len(edu_matches)} patterns)")
    
    # Check 5: No error indicators (20 points)
    error_patterns = [
        r"(?i)i don'?t know",
        r"(?i)i cannot (explain|understand|help)",
        r"(?i)error occurred",
        r"(?i)unable to (process|analyze|explain)",
        r"(?i)not sure what",
        r"(?i)invalid (code|syntax|input)",
    ]
    errors_found = [p for p in error_patterns if re.search(p, output)]
    
    if not errors_found:
        score += 0.2
        checks.append("✓ No error indicators")
    else:
        checks.append(f"✗ Contains error indicators ({len(errors_found)} found)")
    
    # Determine pass/fail (60% threshold)
    passed = score >= 0.6
    
    return {
        "score": score,
        "passed": passed,
        "message": f"Explanation quality: {score:.0%} ({'PASS' if passed else 'FAIL'})",
        "details": {
            "checks": checks,
            "output_length": len(output),
            "language_detected": language,
            "threshold": 0.6,
        },
    }


if __name__ == "__main__":
    # Read context from stdin (for CLI usage)
    context = json.load(sys.stdin)
    result = grade(context)
    print(json.dumps(result, indent=2))
