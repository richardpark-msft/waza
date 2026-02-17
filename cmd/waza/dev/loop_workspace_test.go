package dev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryResolveSkillDir_SingleSkillWorkspace(t *testing.T) {
	dir := t.TempDir()
	skillContent := "---\nname: ws-dev-skill\ndescription: \"A test skill.\"\n---\n# Body\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644))
	t.Chdir(dir)

	result := tryResolveSkillDir("")
	assert.NotEmpty(t, result)
	assert.Equal(t, dir, result)
}

func TestTryResolveSkillDir_ByName(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")

	for _, name := range []string{"one", "two"} {
		skillDir := filepath.Join(skillsDir, name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		content := "---\nname: " + name + "\ndescription: \"desc\"\n---\n# Body\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))
	}
	t.Chdir(dir)

	result := tryResolveSkillDir("two")
	assert.Equal(t, filepath.Join(skillsDir, "two"), result)
}

func TestTryResolveSkillDir_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	result := tryResolveSkillDir("")
	assert.Empty(t, result)
}

func TestTryResolveSkillDir_NameNotFound(t *testing.T) {
	dir := t.TempDir()
	skillContent := "---\nname: existing\ndescription: \"desc\"\n---\n# Body\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644))
	t.Chdir(dir)

	result := tryResolveSkillDir("nonexistent")
	assert.Empty(t, result)
}
