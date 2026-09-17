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
			lsCommand(),
			rmCommand(),
			openCommand(),
			pathCommand(),
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

func lsCommand() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "list worktrees (branch first, then path)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			mainDir, err := git.MainDir()
			if err != nil {
				return err
			}
			return worktree.List(mainDir, os.Stdout)
		},
	}
}

func rmCommand() *cli.Command {
	return &cli.Command{
		Name:      "rm",
		Usage:     "remove a worktree and delete its branch (interactive picker if name omitted)",
		ArgsUsage: "[name]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().Get(0)

			mainDir, err := git.MainDir()
			if err != nil {
				return err
			}

			cfg, err := config.Load(mainDir)
			if err != nil {
				return err
			}

			return worktree.Remove(cfg, mainDir, name, os.Stdout)
		},
	}
}

func openCommand() *cli.Command {
	return &cli.Command{
		Name:      "open",
		Usage:     "open a worktree in the configured editor (interactive picker if name omitted)",
		ArgsUsage: "[name]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().Get(0)

			mainDir, err := git.MainDir()
			if err != nil {
				return err
			}

			cfg, err := config.Load(mainDir)
			if err != nil {
				return err
			}

			return worktree.Open(cfg, mainDir, name, os.Stdout)
		},
	}
}

func pathCommand() *cli.Command {
	return &cli.Command{
		Name:      "path",
		Usage:     "print the worktree path (interactive picker if name omitted)",
		ArgsUsage: "[name]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().Get(0)

			mainDir, err := git.MainDir()
			if err != nil {
				return err
			}

			cfg, err := config.Load(mainDir)
			if err != nil {
				return err
			}

			dir, err := worktree.Path(cfg, mainDir, name)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, dir)
			return nil
		},
	}
}
