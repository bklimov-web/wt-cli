// Package installer runs a worktree's install step: either a user-supplied
// list of commands, or an auto-detected JS package manager's install
// command based on lockfiles present.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// manager pairs a lockfile with the command that installs from it.
type manager struct {
	lockfile string
	command  []string
}

var managers = []manager{
	{"pnpm-lock.yaml", []string{"pnpm", "install"}},
	{"yarn.lock", []string{"yarn", "install", "--frozen-lockfile"}},
	{"bun.lockb", []string{"bun", "install"}},
	{"package-lock.json", []string{"npm", "ci"}},
}

// Install runs the worktree's install step in dir. If commands is
// non-empty, each entry is run in order through the shell (so entries can
// use shell syntax like "&&" or pipes), stopping at the first failure.
// Otherwise it falls back to detecting a JS package manager from dir's
// lockfile. It's a no-op (with a status line) if commands is empty and no
// known lockfile is present.
func Install(dir string, commands []string) error {
	if len(commands) > 0 {
		for _, c := range commands {
			fmt.Printf("wt: running %s\n", c)
			cmd := exec.Command("sh", "-c", c)
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
	}

	for _, m := range managers {
		if _, err := os.Stat(filepath.Join(dir, m.lockfile)); err != nil {
			continue
		}
		fmt.Printf("wt: running %s\n", strings.Join(m.command, " "))
		cmd := exec.Command(m.command[0], m.command[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Println("wt: no lockfile found, skipping install")
	return nil
}
