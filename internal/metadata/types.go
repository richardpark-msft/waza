// Package metadata provides types and helpers for generating azd extension
// metadata from a Cobra command tree.  The types mirror the upstream
// azure-dev/cli/azd/pkg/extensions schema so the JSON output is wire-compatible,
// but are defined locally to avoid pulling in the full azure-dev module and its
// go 1.25 requirement.
package metadata

// ExtensionCommandMetadata represents the complete metadata for an extension.
type ExtensionCommandMetadata struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            string    `json:"id"`
	Commands      []Command `json:"commands"`
}

// Command represents a command or subcommand in the extension's command tree.
type Command struct {
	Name        []string         `json:"name"`
	Short       string           `json:"short"`
	Long        string           `json:"long,omitempty"`
	Usage       string           `json:"usage,omitempty"`
	Examples    []CommandExample `json:"examples,omitempty"`
	Args        []Argument       `json:"args,omitempty"`
	Flags       []Flag           `json:"flags,omitempty"`
	Subcommands []Command        `json:"subcommands,omitempty"`
	Hidden      bool             `json:"hidden,omitempty"`
	Aliases     []string         `json:"aliases,omitempty"`
	Deprecated  string           `json:"deprecated,omitempty"`
}

// CommandExample represents an example usage of a command.
type CommandExample struct {
	Description string `json:"description"`
	Command     string `json:"command"`
}

// Argument represents a positional argument for a command.
type Argument struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Variadic    bool     `json:"variadic,omitempty"`
	ValidValues []string `json:"validValues,omitempty"`
}

// Flag represents a command-line flag/option.
type Flag struct {
	Name        string      `json:"name"`
	Shorthand   string      `json:"shorthand,omitempty"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required,omitempty"`
	ValidValues []string    `json:"validValues,omitempty"`
	Hidden      bool        `json:"hidden,omitempty"`
	Deprecated  string      `json:"deprecated,omitempty"`
}
