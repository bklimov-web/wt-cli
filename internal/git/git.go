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
// "main"), resolved from the local origin/HEAD symref. Returns "" if the
// symref simply isn't set (the common, benign case). err is non-nil only
// for an unexpected failure (e.g. git missing, repo corruption) — distinct
// from "not set" so callers can tell the two apart.
func DefaultBranch(mainDir string) (name string, err error) {
	cmd := exec.Command("git", "-C", mainDir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// --quiet: exit code 1 means the symref is unset, not an error.
			return "", nil
		}
		return "", fmt.Errorf("git symbolic-ref: %w", err)
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"), nil
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
	cmd := exec.Command("git", "-C", mainDir, "worktree", "add", "-b", branch, "--", dir, baseRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DetachedBranch is the sentinel Worktree.Branch value for a worktree
// checked out at a detached HEAD (no branch).
const DetachedBranch = "(detached)"

// Worktree is one entry from `git worktree list`: its checkout path and the
// branch checked out there (DetachedBranch if none).
type Worktree struct {
	Path   string
	Branch string
}

// WorktreeList returns all worktrees registered against mainDir (the main
// checkout included), parsed from `git worktree list --porcelain`.
func WorktreeList(mainDir string) ([]Worktree, error) {
	out, err := exec.Command("git", "-C", mainDir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var (
		entries []Worktree
		path    string
		branch  string
	)
	flush := func() {
		if path != "" {
			entries = append(entries, Worktree{Path: path, Branch: branch})
		}
		path, branch = "", ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case strings.HasPrefix(line, "detached"):
			branch = DetachedBranch
		case line == "":
			flush()
		}
	}
	flush()

	return entries, nil
}

// WorktreeRemove runs `git worktree remove <dir>` in mainDir.
func WorktreeRemove(mainDir, dir string) error {
	cmd := exec.Command("git", "-C", mainDir, "worktree", "remove", "--", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WorktreePrune runs `git worktree prune` in mainDir.
func WorktreePrune(mainDir string) error {
	cmd := exec.Command("git", "-C", mainDir, "worktree", "prune")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BranchDelete runs `git branch -d <branch>` in mainDir.
func BranchDelete(mainDir, branch string) error {
	cmd := exec.Command("git", "-C", mainDir, "branch", "-d", "--", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
