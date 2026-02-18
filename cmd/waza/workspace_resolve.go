package main

import (
	"fmt"
	"os"

	"github.com/spboyer/waza/internal/workspace"
)

// resolveSkillsFromArgs uses workspace detection to resolve skills from CLI args.
// Behavior:
//   - Explicit path to a file (e.g. eval.yaml) → returns nil (caller uses path directly)
//   - Skill name arg + workspace → returns that single skill
//   - No args + single-skill workspace → returns that single skill
//   - No args + multi-skill workspace → returns all skills
//   - No workspace detected → returns error
func resolveSkillsFromArgs(args []string) ([]workspace.SkillInfo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	ctx, err := workspace.DetectContext(wd)
	if err != nil {
		return nil, fmt.Errorf("detecting workspace: %w", err)
	}

	if len(args) > 0 {
		arg := args[0]
		// If arg looks like a file path (has extension or separator), treat as explicit path
		if workspace.LooksLikePath(arg) {
			return nil, nil // caller handles explicit path
		}
		// Treat as skill name
		if ctx.Type == workspace.ContextNone {
			return nil, fmt.Errorf("no workspace detected and %q is not a file path", arg)
		}
		si, err := workspace.FindSkill(ctx, arg)
		if err != nil {
			return nil, err
		}
		return []workspace.SkillInfo{*si}, nil
	}

	// No args — use workspace detection
	switch ctx.Type {
	case workspace.ContextSingleSkill, workspace.ContextMultiSkill:
		return ctx.Skills, nil
	default:
		return nil, fmt.Errorf("no skills detected in workspace; provide a path or skill name")
	}
}

// resolveEvalPath finds eval.yaml for a skill using workspace detection.
func resolveEvalPath(si *workspace.SkillInfo) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	ctx, err := workspace.DetectContext(wd)
	if err != nil {
		return "", fmt.Errorf("detecting workspace: %w", err)
	}
	// Ensure the skill is in the context so FindEval can locate it
	found := false
	for _, s := range ctx.Skills {
		if s.Name == si.Name {
			found = true
			break
		}
	}
	if !found {
		ctx.Skills = append(ctx.Skills, *si)
	}
	evalPath, err := workspace.FindEval(ctx, si.Name)
	if err != nil {
		return "", err
	}
	if evalPath == "" {
		return "", fmt.Errorf("no eval.yaml found for skill %q", si.Name)
	}
	return evalPath, nil
}
