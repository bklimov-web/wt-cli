// Package picker provides interactive terminal prompts (selection, yes/no
// confirmation) shared by wt commands that fall back to a picker when the
// user doesn't name a worktree explicitly.
package picker

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"wt/internal/git"
)

// SelectWorktree prompts the user to choose one of entries, showing each
// one's branch as the label — with a "(main)" marker for the entry whose
// path is mainDir, so callers that include the main checkout among entries
// can still tell it apart. Returns an error if entries is empty.
func SelectWorktree(action string, entries []git.Worktree, mainDir string) (git.Worktree, error) {
	if len(entries) == 0 {
		return git.Worktree{}, fmt.Errorf("no worktrees to %s", action)
	}

	options := make([]huh.Option[int], len(entries))
	for i, e := range entries {
		label := e.Branch
		if e.Path == mainDir {
			label += " (main)"
		}
		options[i] = huh.NewOption(label, i)
	}

	var choice int
	sel := huh.NewSelect[int]().
		Title(fmt.Sprintf("Select a worktree to %s:", action)).
		Options(options...).
		Value(&choice)

	if err := sel.Run(); err != nil {
		return git.Worktree{}, err
	}
	return entries[choice], nil
}

// Confirm asks a yes/no question and returns the answer, defaulting to "No".
func Confirm(title string) (bool, error) {
	var ok bool
	c := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&ok)

	if err := c.Run(); err != nil {
		return false, err
	}
	return ok, nil
}
