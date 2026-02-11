package main

import (
	"encoding/json"
	"fmt"

	"github.com/spboyer/waza/internal/metadata"
	"github.com/spf13/cobra"
)

const metadataSchemaVersion = "1.0"
const extensionID = "microsoft.azd.waza"

func newMetadataCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Short:  "Output extension metadata as JSON",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			meta := metadata.GenerateExtensionMetadata(metadataSchemaVersion, extensionID, rootCmd)

			data, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			data = append(data, '\n')
			_, err = cmd.OutOrStdout().Write(data)
			if err != nil {
				return fmt.Errorf("failed to write metadata: %w", err)
			}
			return nil
		},
	}
}
