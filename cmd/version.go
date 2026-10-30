package cmd

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the lazydash version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "lazydash "+version.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
