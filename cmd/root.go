package cmd

import (
	"fmt"
	"os"

	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/tui"
	"github.com/spf13/cobra"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "lazydash",
	Short: "A terminal UI for GitHub Projects and local git-native issues",
	Long: `lazydash is a keyboard-driven terminal UI for browsing and managing issues:
GitHub Projects boards and local git-native issues (stored in the repo under
refs/issues/). Run it inside a repository to work with local issues fully
offline, or anywhere to browse and edit your GitHub Projects.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		return tui.Start(cfg)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "path to config file")
}
