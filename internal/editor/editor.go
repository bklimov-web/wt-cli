// Package editor opens a worktree directory in the configured editor.
package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// Open runs the editor command with dir as its argument. It errors if the
// command isn't found on PATH.
func Open(editorCmd, dir string) error {
	if _, err := exec.LookPath(editorCmd); err != nil {
		return fmt.Errorf("'%s' not found in PATH", editorCmd)
	}
	cmd := exec.Command(editorCmd, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
