package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillMD_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\nname: test-skill\ndescription: A test skill.\n---\n\n# Body\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	fm, err := ParseSkillMD(path)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", fm.Name)
	assert.Equal(t, "A test skill.", fm.Description)
}

func TestParseSkillMD_MultilineDescription(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\nname: multi\ndescription: |\n  Line one.\n  Line two.\n---\n\n# Body\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	fm, err := ParseSkillMD(path)
	require.NoError(t, err)
	assert.Equal(t, "multi", fm.Name)
	assert.Contains(t, fm.Description, "Line one.")
	assert.Contains(t, fm.Description, "Line two.")
}

func TestParseSkillMD_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\ndescription: No name here\n---\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := ParseSkillMD(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required 'name'")
}

func TestParseSkillMD_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "# Just a markdown file\nNo frontmatter here.\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := ParseSkillMD(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing YAML frontmatter delimiter")
}

func TestParseSkillMD_MissingClosingDelimiter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\nname: broken\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := ParseSkillMD(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing closing frontmatter delimiter")
}

func TestParseSkillMD_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	require.NoError(t, os.WriteFile(path, []byte(""), 0644))

	_, err := ParseSkillMD(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseSkillMD_FileNotFound(t *testing.T) {
	_, err := ParseSkillMD("/nonexistent/SKILL.md")
	assert.Error(t, err)
}

func TestParseSkillMD_PathTraversalInName(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
	}{
		{"forward slash", "foo/bar"},
		{"backslash", "foo\\bar"},
		{"dot-dot", "foo..bar"},
		{"parent traversal", "../etc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "SKILL.md")
			content := fmt.Sprintf("---\nname: %s\ndescription: bad\n---\n\n# Skill\n", tt.skillName)
			require.NoError(t, os.WriteFile(path, []byte(content), 0644))

			_, err := ParseSkillMD(path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid path characters")
		})
	}
}
