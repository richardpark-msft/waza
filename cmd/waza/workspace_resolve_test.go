package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spboyer/waza/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLooksLikePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"eval.yaml", true},
		{"./skills/foo", true},
		{"skills/foo", true},
		{".", true},
		{"code-explainer", false},
		{"my-skill", false},
		{"foo.yaml", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, workspace.LooksLikePath(tt.input))
		})
	}
}

func TestResolveSkillsFromArgs_ExplicitPath(t *testing.T) {
	// Explicit paths should return nil (caller handles directly)
	skills, err := resolveSkillsFromArgs([]string{"eval.yaml"})
	assert.NoError(t, err)
	assert.Nil(t, skills)

	skills, err = resolveSkillsFromArgs([]string{"./my-skill"})
	assert.NoError(t, err)
	assert.Nil(t, skills)

	skills, err = resolveSkillsFromArgs([]string{"."})
	assert.NoError(t, err)
	assert.Nil(t, skills)
}

func TestResolveSkillsFromArgs_SingleSkillWorkspace(t *testing.T) {
	dir := t.TempDir()
	skillContent := `---
name: test-skill
description: "A test skill."
---
# Test
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644))
	t.Chdir(dir)

	skills, err := resolveSkillsFromArgs(nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "test-skill", skills[0].Name)
}

func TestResolveSkillsFromArgs_MultiSkillWorkspace(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")

	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(skillsDir, name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		content := "---\nname: " + name + "\ndescription: \"desc\"\n---\n# Body\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))
	}
	t.Chdir(dir)

	skills, err := resolveSkillsFromArgs(nil)
	require.NoError(t, err)
	require.Len(t, skills, 2)

	names := []string{skills[0].Name, skills[1].Name}
	assert.Contains(t, names, "skill-a")
	assert.Contains(t, names, "skill-b")
}

func TestResolveSkillsFromArgs_SkillName(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")

	for _, name := range []string{"alpha", "beta"} {
		skillDir := filepath.Join(skillsDir, name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		content := "---\nname: " + name + "\ndescription: \"desc\"\n---\n# Body\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))
	}
	t.Chdir(dir)

	skills, err := resolveSkillsFromArgs([]string{"alpha"})
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "alpha", skills[0].Name)
}

func TestResolveSkillsFromArgs_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	_, err := resolveSkillsFromArgs(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no skills detected")
}

func TestResolveEvalPath_WithColocatedEval(t *testing.T) {
	dir := t.TempDir()
	skillContent := "---\nname: test-skill\ndescription: \"desc\"\n---\n# Body\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eval.yaml"), []byte("name: test\n"), 0o644))
	t.Chdir(dir)

	si := workspace.SkillInfo{Name: "test-skill", Dir: dir, SkillPath: filepath.Join(dir, "SKILL.md")}
	evalPath, err := resolveEvalPath(&si)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "eval.yaml"), evalPath)
}
