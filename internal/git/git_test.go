package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initRepo creates a fresh git repo with one commit in a temp dir and
// returns its path, resolved through any symlinks (e.g. macOS's
// /var -> /private/var) so it matches what `git worktree list` reports.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	run("commit", "--allow-empty", "-q", "-m", "init")

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestDefaultBranch_NoOrigin(t *testing.T) {
	dir := initRepo(t)

	name, err := DefaultBranch(dir)
	if err != nil {
		t.Fatalf("DefaultBranch() error = %v, want nil (unset symref is not an error)", err)
	}
	if name != "" {
		t.Fatalf("DefaultBranch() = %q, want empty (no origin configured)", name)
	}
}

func TestDefaultBranch_UnexpectedFailure(t *testing.T) {
	// A directory that isn't a git repo at all triggers a real exec
	// failure, not just "symref unset".
	dir := t.TempDir()

	_, err := DefaultBranch(dir)
	if err == nil {
		t.Fatal("DefaultBranch() error = nil, want non-nil for a non-git directory")
	}
}

func TestWorktreeAddAndRemove_RoundTrip(t *testing.T) {
	mainDir := initRepo(t)
	branch := "feature/x"
	dir := mainDir + "-wt"

	if err := WorktreeAdd(mainDir, dir, branch, "HEAD"); err != nil {
		t.Fatalf("WorktreeAdd() error = %v", err)
	}

	entries, err := WorktreeList(mainDir)
	if err != nil {
		t.Fatalf("WorktreeList() error = %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Path == dir && e.Branch == branch {
			found = true
		}
	}
	if !found {
		t.Fatalf("WorktreeList() = %+v, want an entry for %s/%s", entries, dir, branch)
	}

	if err := WorktreeRemove(mainDir, dir, false); err != nil {
		t.Fatalf("WorktreeRemove() error = %v", err)
	}
	if err := BranchDelete(mainDir, branch); err != nil {
		t.Fatalf("BranchDelete() error = %v", err)
	}
}

// TestWorktreeAddAndRemove_LeadingDashDir confirms the "--" separator in
// WorktreeAdd and WorktreeRemove stops git from treating a dash-prefixed
// worktree path as a flag. Unlike branch names (which git's
// check-ref-format rejects outright if they start with "-"), a worktree
// path is an arbitrary filesystem path and can legitimately start with
// "-", so this is the argument position that's actually exploitable.
func TestWorktreeAddAndRemove_LeadingDashDir(t *testing.T) {
	mainDir := initRepo(t)
	branch := "feature/y"
	dir := filepath.Join(mainDir+"-parent", "-oddname")

	if err := WorktreeAdd(mainDir, dir, branch, "HEAD"); err != nil {
		t.Fatalf("WorktreeAdd() error = %v", err)
	}
	if err := WorktreeRemove(mainDir, dir, false); err != nil {
		t.Fatalf("WorktreeRemove() error = %v", err)
	}
}

func TestWorktreeList_DetachedHead(t *testing.T) {
	mainDir := initRepo(t)
	dir := mainDir + "-detached"

	cmd := exec.Command("git", "-C", mainDir, "worktree", "add", "--detach", "--", dir, "HEAD")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add --detach: %v\n%s", err, out)
	}

	entries, err := WorktreeList(mainDir)
	if err != nil {
		t.Fatalf("WorktreeList() error = %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Path == dir {
			if e.Branch != DetachedBranch {
				t.Fatalf("Branch = %q, want %q", e.Branch, DetachedBranch)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("WorktreeList() = %+v, want an entry for %s", entries, dir)
	}
}

func TestStatusAndUnpushed(t *testing.T) {
	dir := initRepo(t)

	status, err := StatusLines(dir)
	if err != nil || len(status) != 0 {
		t.Fatalf("clean repo: StatusLines() = %v, %v", status, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err = StatusLines(dir)
	if err != nil || len(status) != 1 || status[0] != "?? new.txt" {
		t.Fatalf("StatusLines() = %v, %v; want [?? new.txt]", status, err)
	}

	// No remotes at all: the init commit counts as unpushed.
	unpushed, err := UnpushedCommits(dir)
	if err != nil || len(unpushed) != 1 {
		t.Fatalf("UnpushedCommits() = %v, %v; want 1 commit", unpushed, err)
	}

	// Once a remote-tracking ref covers HEAD, nothing is unpushed.
	if out, err := exec.Command("git", "-C", dir, "update-ref", "refs/remotes/origin/main", "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
	unpushed, err = UnpushedCommits(dir)
	if err != nil || len(unpushed) != 0 {
		t.Fatalf("UnpushedCommits() = %v, %v; want none", unpushed, err)
	}
}
