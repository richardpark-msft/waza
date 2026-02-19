package main

import (
	"fmt"
	"os"

	"github.com/spboyer/waza/internal/generate"
	"github.com/spboyer/waza/internal/workspace"
	"github.com/spf13/cobra"
)

var generateOutputDir string

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <skill-name | SKILL.md>",
		Short: "Generate an eval suite from a SKILL.md file (alias for 'waza new')",
		Long: `Generate evaluation files from a SKILL.md file.

Note: 'waza generate' is an alias for 'waza new'.

Parses the YAML frontmatter (name, description) from the given SKILL.md and
creates an eval.yaml, starter task files, and a fixtures directory using the
same idempotent scaffolding as 'waza new'.

If the argument looks like a skill name (no path separators or file extension),
it is resolved via workspace detection to find the SKILL.md path.

When --output-dir is specified, scaffolding is written into that directory
instead of the current working directory.`,
		Args: cobra.ExactArgs(1),
		RunE: generateCommandE,
	}

	cmd.Flags().StringVarP(&generateOutputDir, "output-dir", "d", "", "Directory to scaffold into (default: current directory)")

	return cmd
}

func generateCommandE(cmd *cobra.Command, args []string) error {
	skillPath := args[0]

	fmt.Fprintln(cmd.OutOrStdout(), "Note: 'waza generate' is an alias for 'waza new'.") //nolint:errcheck

	// If arg looks like a skill name (not a path), resolve via workspace
	if !workspace.LooksLikePath(skillPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		ctx, err := workspace.DetectContext(wd)
		if err != nil {
			return fmt.Errorf("detecting workspace: %w", err)
		}
		si, err := workspace.FindSkill(ctx, skillPath)
		if err != nil {
			return err
		}
		skillPath = si.SkillPath
	}

	skill, err := generate.ParseSkillMD(skillPath)
	if err != nil {
		return fmt.Errorf("failed to parse SKILL.md: %w", err)
	}

	// If --output-dir is specified, chdir so scaffolding writes there
	if generateOutputDir != "" {
		if err := os.MkdirAll(generateOutputDir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		origDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		if err := os.Chdir(generateOutputDir); err != nil {
			return fmt.Errorf("changing to output directory: %w", err)
		}
		defer os.Chdir(origDir) //nolint:errcheck
	}

	// Delegate to waza new's unified code path
	return newCommandE(cmd, []string{skill.Name}, "")
}
