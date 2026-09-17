package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"wt/internal/config"
	"wt/internal/git"
	"wt/internal/worktree"
)

func main() {
	cmd := &cli.Command{
		Name:  "wt",
		Usage: "create git worktrees with local files and deps in place",
		Commands: []*cli.Command{
			newCommand(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "wt: %v\n", err)
		os.Exit(1)
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:      "new",
		Usage:     "create a worktree from origin's default branch",
		ArgsUsage: "<name> [branch]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().Get(0)
			branch := cmd.Args().Get(1)
			if name == "" {
				return fmt.Errorf("name required")
			}

			mainDir, err := git.MainDir()
			if err != nil {
				return err
			}

			cfg, err := config.Load(mainDir)
			if err != nil {
				return err
			}

			_, err = worktree.New(cfg, mainDir, name, branch, os.Stdout)
			return err
		},
	}
}
