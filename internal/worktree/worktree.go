// Package worktree orchestrates creating a new git worktree: resolving
// paths from config templates, branching from origin's default branch,
// copying env files, and installing dependencies.
package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"wt/internal/config"
	"wt/internal/editor"
	"wt/internal/git"
	"wt/internal/installer"
)

// Dir resolves the base directory (parent of all worktrees for this repo)
// from cfg.WorktreeDir, substituting {repo} and resolving it relative to
// mainDir.
func Dir(cfg config.Config, mainDir string) string {
	repo := filepath.Base(mainDir)
	template := strings.ReplaceAll(cfg.WorktreeDir, "{repo}", repo)
	return filepath.Clean(filepath.Join(mainDir, template))
}

// Branch substitutes {name} into cfg.BranchPattern.
func Branch(cfg config.Config, name string) string {
	return strings.ReplaceAll(cfg.BranchPattern, "{name}", name)
}

// New creates a worktree named name (branch defaults to cfg.BranchPattern
// with name substituted if branch is empty), branching from origin's
// default branch. It copies configured env files, runs the install
// command unless disabled, and opens the editor unless disabled. Status
// messages are written to w. It returns the created worktree's path.
func New(cfg config.Config, mainDir, name, branch string, w io.Writer) (string, error) {
	dir := filepath.Join(Dir(cfg, mainDir), name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("%s already exists", dir)
	}

	if branch == "" {
		branch = Branch(cfg, name)
	}

	def := git.DefaultBranch(mainDir)
	if def == "" {
		def = "master"
	}

	fmt.Fprintln(w, "wt: fetching origin...")
	if err := git.Fetch(mainDir); err != nil {
		fmt.Fprintf(w, "wt: WARNING fetch failed — branching from your local origin/%s\n", def)
	}

	baseRef := "origin/" + def
	if !git.RefExists(mainDir, baseRef) {
		return "", fmt.Errorf("no ref %s — is the remote set up?", baseRef)
	}

	if err := git.WorktreeAdd(mainDir, dir, branch, baseRef); err != nil {
		return "", err
	}

	for _, f := range cfg.EnvFiles {
		src := filepath.Join(mainDir, f)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return dir, fmt.Errorf("reading %s: %w", f, err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), data, 0o644); err != nil {
			return dir, fmt.Errorf("copying %s: %w", f, err)
		}
		fmt.Fprintf(w, "wt: copied %s\n", f)
	}

	if !cfg.NoInstall {
		if err := installer.Install(dir); err != nil {
			return dir, fmt.Errorf("install failed: %w", err)
		}
	}

	fmt.Fprintf(w, "\nwt: %s  ->  %s\n", branch, dir)

	if !cfg.NoCode {
		if err := editor.Open(cfg.Editor, dir); err != nil {
			fmt.Fprintf(w, "wt: WARNING could not open editor: %v\n", err)
		}
	}

	return dir, nil
}
