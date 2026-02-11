package metadata

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// GenerateExtensionMetadata builds ExtensionCommandMetadata from a Cobra root
// command.  It is the local equivalent of azdext.GenerateExtensionMetadata.
func GenerateExtensionMetadata(schemaVersion, id string, root *cobra.Command) *ExtensionCommandMetadata {
	return &ExtensionCommandMetadata{
		SchemaVersion: schemaVersion,
		ID:            id,
		Commands:      generateCommands(root),
	}
}

func generateCommands(cmd *cobra.Command) []Command {
	var commands []Command
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "" {
			continue
		}
		command := generateCommand(subCmd)
		if len(command.Name) == 0 {
			continue
		}
		commands = append(commands, command)
	}
	return commands
}

func generateCommand(cmd *cobra.Command) Command {
	path := buildCommandPath(cmd)
	command := Command{
		Name:     path,
		Short:    cmd.Short,
		Long:     cmd.Long,
		Usage:    cmd.UseLine(),
		Examples: generateExamples(cmd),
		Args:     generateArgs(cmd),
		Flags:    generateFlags(cmd),
		Hidden:   cmd.Hidden,
		Aliases:  cmd.Aliases,
	}
	if cmd.Deprecated != "" {
		command.Deprecated = cmd.Deprecated
	}
	if cmd.HasSubCommands() {
		command.Subcommands = generateCommands(cmd)
	}
	return command
}

func buildCommandPath(cmd *cobra.Command) []string {
	var path []string
	current := cmd
	for current != nil && current.Use != "" {
		use := current.Use
		name := use
		for i, r := range use {
			if r == ' ' || r == '\t' {
				name = use[:i]
				break
			}
		}
		path = append([]string{name}, path...)
		current = current.Parent()
	}
	if len(path) > 0 {
		path = path[1:]
	}
	return path
}

func generateExamples(cmd *cobra.Command) []CommandExample {
	if cmd.Example == "" {
		return nil
	}
	return []CommandExample{
		{
			Description: "Usage example",
			Command:     cmd.Example,
		},
	}
}

func generateArgs(_ *cobra.Command) []Argument {
	return nil
}

func generateFlags(cmd *cobra.Command) []Flag {
	cmd.InitDefaultHelpFlag()
	var flags []Flag
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagMeta := Flag{
			Name:        flag.Name,
			Shorthand:   flag.Shorthand,
			Description: flag.Usage,
			Type:        getFlagType(flag),
			Hidden:      flag.Hidden,
		}
		if flag.DefValue != "" {
			flagMeta.Default = flag.DefValue
		}
		if flag.Deprecated != "" {
			flagMeta.Deprecated = flag.Deprecated
		}
		flags = append(flags, flagMeta)
	})
	return flags
}

func getFlagType(flag *pflag.Flag) string {
	switch flag.Value.Type() {
	case "bool":
		return "bool"
	case "int", "int32", "int64":
		return "int"
	case "string":
		return "string"
	case "stringSlice", "stringArray":
		return "stringArray"
	case "intSlice":
		return "intArray"
	default:
		return "string"
	}
}
