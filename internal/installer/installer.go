// Package installer auto-detects a JS package manager from lockfiles and
// runs its install command.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// Install detects the package manager from dir's lockfile and runs its
// install command there. It's a no-op (with a status line) if no known
// lockfile is present.
func Install(dir string) error {
	for _, m := range managers {
		if _, err := os.Stat(filepath.Join(dir, m.lockfile)); err != nil {
			continue
		}
		fmt.Printf("wt: running %s\n", joinCmd(m.command))
		cmd := exec.Command(m.command[0], m.command[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Println("wt: no lockfile found, skipping install")
	return nil
}

func joinCmd(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += " " + p
	}
	return out
}
