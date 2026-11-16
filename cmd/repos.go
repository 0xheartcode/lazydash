package cmd

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/repostore"
	"github.com/0xheartcode/lazydash/internal/source/local"
	"github.com/spf13/cobra"
)

var addRepoCmd = &cobra.Command{
	Use:   "add-repo [path]",
	Short: "Register a repository to show on the dashboard (defaults to the current directory)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) == 1 {
			path = args[0]
		}
		root, err := local.RepoRoot(path)
		if err != nil {
			return fmt.Errorf("%q is not inside a git repository", path)
		}
		added, stored, err := repostore.Add(root)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if !added {
			fmt.Fprintf(out, "already registered: %s\n", stored)
			return nil
		}
		fmt.Fprintf(out, "registered: %s\n", stored)
		if !local.HasIssues(root) {
			fmt.Fprintln(out, "note: no local issues yet (refs/issues/); it will show empty until you create one with gli")
		}
		return nil
	},
}

var rmRepoCmd = &cobra.Command{
	Use:   "rm-repo [path]",
	Short: "Unregister a repository from the dashboard (defaults to the current directory)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) == 1 {
			path = args[0]
		}
		// Resolve to the repo root when possible, so `rm-repo .` matches what
		// add-repo stored; fall back to the raw path when the repo is gone.
		target := path
		if root, err := local.RepoRoot(path); err == nil {
			target = root
		}
		removed, stored, err := repostore.Remove(target)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if !removed {
			fmt.Fprintf(out, "not registered: %s\n", stored)
			return nil
		}
		fmt.Fprintf(out, "unregistered: %s\n", stored)
		return nil
	},
}

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List the repositories registered with add-repo",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repostore.Load()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(repos) == 0 {
			fmt.Fprintln(out, "no repositories registered (use `lazydash add-repo`)")
			return nil
		}
		for _, r := range repos {
			note := ""
			if !local.HasIssues(r) {
				note = "  (no issues yet)"
			}
			fmt.Fprintf(out, "%s%s\n", r, note)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addRepoCmd, rmRepoCmd, reposCmd)
}
