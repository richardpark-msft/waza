package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newInitCommand() *cobra.Command {
	var interactive bool
	var noSkill bool

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a waza project",
		Long: `Initialize a waza project with the required directory structure.

Idempotently ensures the project has:
  - skills/         Skill definitions directory
  - evals/          Evaluation suites directory
  - .github/workflows/eval.yml  CI pipeline
  - .gitignore      With waza-specific entries
  - README.md       Getting started guide

Only creates what's missing — never overwrites existing files.

After scaffolding, prompts to create your first skill (calls waza new internally).

Use --no-skill to skip the skill creation prompt.
Use --interactive for project-level wizard (reserved for future use).

If no directory is specified, the current directory is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return initCommandE(cmd, args, interactive, noSkill)
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run project-level setup wizard")
	cmd.Flags().BoolVar(&noSkill, "no-skill", false, "Skip the first-skill creation prompt")

	return cmd
}

func initCommandE(cmd *cobra.Command, args []string, interactive bool, noSkill bool) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	out := cmd.OutOrStdout()
	projectName := filepath.Base(absOrDefault(dir))

	if interactive {
		fmt.Fprintln(out, "Note: interactive project setup coming soon. Using defaults.") //nolint:errcheck
	}
	fmt.Fprintf(out, "Initializing waza project in %s\n\n", dir) //nolint:errcheck

	// Ensure required directories
	for _, d := range []string{
		filepath.Join(dir, "skills"),
		filepath.Join(dir, "evals"),
	} {
		status, err := ensureDir(d)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "  %s %s\n", status, d) //nolint:errcheck
	}

	// Ensure required files — .waza.yaml needs special handling for prompts
	wazaConfigPath := filepath.Join(dir, ".waza.yaml")
	_, wazaExists := os.Stat(wazaConfigPath)

	needConfigPrompt := wazaExists != nil
	needSkillPrompt := !noSkill

	// Collect all prompt values
	var engine, model, skillName string
	var createSkill bool
	engine = "copilot-sdk"
	model = "claude-sonnet-4.6"

	// Use interactive (arrow-key) mode when running in a terminal,
	// fall back to accessible (number-entry) mode for pipes/CI.
	accessible := !term.IsTerminal(int(os.Stdin.Fd()))

	// Build form with one question per group (shown one at a time)
	var groups []*huh.Group

	if needConfigPrompt {
		// Group 1: Engine selection
		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Default evaluation engine").
				Description("Choose how evals are executed").
				Options(
					huh.NewOption("Copilot SDK — real model execution", "copilot-sdk"),
					huh.NewOption("Mock — fast iteration, no API calls", "mock"),
				).
				Value(&engine),
		))

		// Group 2: Model selection (only shown when engine is copilot-sdk)
		// Note: Copilot SDK (v0.1.22) has no model enumeration API.
		// Update this list as new models become available.
		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Default model").
				Description("Model used for evaluations").
				Options(
					huh.NewOption("claude-sonnet-4.6", "claude-sonnet-4.6"),
					huh.NewOption("claude-sonnet-4.5", "claude-sonnet-4.5"),
					huh.NewOption("claude-haiku-4.5", "claude-haiku-4.5"),
					huh.NewOption("claude-opus-4.6", "claude-opus-4.6"),
					huh.NewOption("claude-opus-4.6-fast", "claude-opus-4.6-fast"),
					huh.NewOption("claude-opus-4.5", "claude-opus-4.5"),
					huh.NewOption("claude-sonnet-4", "claude-sonnet-4"),
					huh.NewOption("gemini-3-pro-preview", "gemini-3-pro-preview"),
					huh.NewOption("gpt-5.3-codex", "gpt-5.3-codex"),
					huh.NewOption("gpt-5.2-codex", "gpt-5.2-codex"),
					huh.NewOption("gpt-5.2", "gpt-5.2"),
					huh.NewOption("gpt-5.1-codex-max", "gpt-5.1-codex-max"),
					huh.NewOption("gpt-5.1-codex", "gpt-5.1-codex"),
					huh.NewOption("gpt-5.1", "gpt-5.1"),
					huh.NewOption("gpt-5", "gpt-5"),
					huh.NewOption("gpt-5.1-codex-mini", "gpt-5.1-codex-mini"),
					huh.NewOption("gpt-5-mini", "gpt-5-mini"),
					huh.NewOption("gpt-4.1", "gpt-4.1"),
				).
				Value(&model),
		).WithHideFunc(func() bool {
			return engine != "copilot-sdk"
		}))
	}

	if needSkillPrompt {
		// Group 3: Create first skill?
		groups = append(groups, huh.NewGroup(
			huh.NewConfirm().
				Title("Create your first skill?").
				Affirmative("Yes").
				Negative("No").
				Value(&createSkill),
		))

		// Group 4: Skill name (only shown if user said yes)
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("Skill name").
				Description("A kebab-case name for your skill (e.g. azure-deploy)").
				Placeholder("my-skill").
				Value(&skillName).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("skill name is required")
					}
					if strings.ContainsAny(s, `/\ `) {
						return fmt.Errorf("skill name cannot contain spaces or path separators")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return !createSkill
		}))
	}

	if len(groups) > 0 {
		form := huh.NewForm(groups...).
			WithAccessible(accessible).
			WithInput(cmd.InOrStdin()).
			WithOutput(cmd.OutOrStdout())

		if err := form.Run(); err != nil {
			// Non-interactive — use defaults, skip skill creation
			engine = "copilot-sdk"
			model = "claude-sonnet-4.6"
			createSkill = false
		}
	}

	// Write .waza.yaml if needed
	if needConfigPrompt {
		content := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/spboyer/waza/main/schemas/waza-config.schema.json
# Waza project configuration
# These defaults are used by 'waza new' when generating eval.yaml files
# and by 'waza run' as fallback values when not specified in eval.yaml.
defaults:
  engine: %s
  model: %s
`, engine, model)
		if err := os.MkdirAll(filepath.Dir(wazaConfigPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(wazaConfigPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	wazaStatus := "✓ exists "
	if needConfigPrompt {
		wazaStatus = "✅ created"
	}
	fmt.Fprintf(out, "  %s %s\n", wazaStatus, wazaConfigPath) //nolint:errcheck

	fileEntries := []struct {
		path    string
		content string
	}{
		{filepath.Join(dir, ".github", "workflows", "eval.yml"), initCIWorkflow()},
		{filepath.Join(dir, ".gitignore"), initGitignore()},
		{filepath.Join(dir, "README.md"), initReadme(projectName)},
	}

	for _, f := range fileEntries {
		status, err := ensureFile(f.path, f.content)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "  %s %s\n", status, f.path) //nolint:errcheck
	}

	fmt.Fprintln(out) //nolint:errcheck

	// Create first skill if requested
	skillName = strings.TrimSpace(skillName)
	if createSkill && skillName != "" {
		origDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory: %w", err)
		}
		if err := os.Chdir(absDir); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}
		defer os.Chdir(origDir) //nolint:errcheck

		fmt.Fprintln(cmd.OutOrStdout()) //nolint:errcheck
		return newCommandE(cmd, []string{skillName}, false, "")
	}

	return nil
}

// ensureDir creates a directory if it doesn't exist and returns a status indicator.
func ensureDir(path string) (string, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return "✓ exists", nil
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return "✅ created", nil
}

// ensureFile creates a file with content if it doesn't exist.
// Parent directories are created as needed.
func ensureFile(path, content string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return "✓ exists", nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", path, err)
	}
	return "✅ created", nil
}

// absOrDefault returns the absolute path, falling back to the input on error.
func absOrDefault(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// --- Init template content ---

func initCIWorkflow() string {
	return `name: Run Skill Evaluations

on:
  pull_request:
    branches: [main]
    paths:
      - 'evals/**'
      - 'skills/**'

permissions:
  contents: read

jobs:
  eval:
    name: Run Evaluations
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Install waza
        run: go install github.com/spboyer/waza/cmd/waza@latest
      - name: Run evaluations
        run: waza run
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: eval-results
          path: results.json
          retention-days: 30
`
}

func initGitignore() string {
	return `results.json
.waza-cache/
coverage.txt
*.exe
`
}

func initReadme(projectName string) string {
	return fmt.Sprintf(`# %s

## Getting Started

1. Create a new skill:
   `+"`"+``+"`"+``+"`"+`bash
   waza new my-skill
   `+"`"+``+"`"+``+"`"+`

2. Edit your skill:
   - Update `+"`"+`skills/my-skill/SKILL.md`+"`"+` with your skill definition
   - Customize eval tasks in `+"`"+`evals/my-skill/tasks/`+"`"+`
   - Add test fixtures to `+"`"+`evals/my-skill/fixtures/`+"`"+`

3. Run evaluations:
   `+"`"+``+"`"+``+"`"+`bash
   waza run                    # run all evals
   waza run my-skill           # run one skill's evals
   `+"`"+``+"`"+``+"`"+`

4. Check compliance:
   `+"`"+``+"`"+``+"`"+`bash
   waza check                  # check all skills
   waza dev my-skill           # improve with real-time scoring
   `+"`"+``+"`"+``+"`"+`

5. Push to trigger CI:
   `+"`"+``+"`"+``+"`"+`bash
   git push
   `+"`"+``+"`"+``+"`"+`
`, projectName)
}
