// Package git wraps the git plumbing commands wt shells out to.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MainDir resolves the main checkout root, even when the current directory
// is inside a linked worktree. It shells out to
// `git rev-parse --git-common-dir` and returns the parent of that dir.
func MainDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	common := strings.TrimSpace(string(out))
	if !filepath.IsAbs(common) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		common = filepath.Join(wd, common)
	}
	return filepath.Dir(common), nil
}

// DefaultBranch returns the short name of origin's default branch (e.g.
// "main"), resolved from the local origin/HEAD symref. Empty if unset.
func DefaultBranch(mainDir string) string {
	out, err := exec.Command("git", "-C", mainDir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")
}

// Fetch runs `git fetch origin` in mainDir.
func Fetch(mainDir string) error {
	cmd := exec.Command("git", "-C", mainDir, "fetch", "origin", "--quiet")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RefExists reports whether ref resolves to a valid object in mainDir.
func RefExists(mainDir, ref string) bool {
	return exec.Command("git", "-C", mainDir, "rev-parse", "--verify", "--quiet", ref).Run() == nil
}

// WorktreeAdd runs `git worktree add <dir> -b <branch> <baseRef>` in mainDir.
func WorktreeAdd(mainDir, dir, branch, baseRef string) error {
	cmd := exec.Command("git", "-C", mainDir, "worktree", "add", dir, "-b", branch, baseRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
