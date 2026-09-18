package worktree

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bklimov-web/wt-cli/internal/config"
	"github.com/bklimov-web/wt-cli/internal/git"
)

// initRepo creates a fresh git repo with one commit and a local
// "origin/<branch>" ref (so New can branch off it without a real remote),
// and returns its path, resolved through any symlinks (e.g. macOS's
// /var -> /private/var) so it matches what `git worktree list` reports.
func initRepo(t *testing.T, defaultBranch string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", defaultBranch)
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	run("commit", "--allow-empty", "-q", "-m", "init")
	// Fake a remote-tracking ref without a real remote, and disable
	// fetch: New calls git.Fetch(mainDir), which needs "origin" to
	// exist. Point it at the repo itself.
	run("remote", "add", "origin", dir)
	run("update-ref", "refs/remotes/origin/"+defaultBranch, "HEAD")
	run("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/"+defaultBranch)

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestDirAndBranch(t *testing.T) {
	cfg := config.Default()
	mainDir := "/repos/myrepo"

	if got, want := Dir(cfg, mainDir), "/repos/myrepo-wt"; got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
	if got, want := Branch(cfg, "foo"), "feature/foo"; got != want {
		t.Errorf("Branch() = %q, want %q", got, want)
	}
}

func TestNew_CopiesEnvFilePreservingPermissions(t *testing.T) {
	mainDir := initRepo(t, "main")
	cfg := config.Default()
	cfg.EnvFiles = []string{".env"}
	cfg.NoInstall = true
	cfg.NoCode = true

	envPath := filepath.Join(mainDir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	dir, err := New(cfg, mainDir, "myfeature", "", &out)
	if err != nil {
		t.Fatalf("New() error = %v\noutput:\n%s", err, out.String())
	}

	copied := filepath.Join(dir, ".env")
	info, err := os.Stat(copied)
	if err != nil {
		t.Fatalf("copied .env missing: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("copied .env perm = %o, want %o", perm, 0o600)
	}

	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "SECRET=1\n" {
		t.Errorf("copied .env content = %q, want %q", data, "SECRET=1\n")
	}
}

func TestNew_InstallFailureWarnsButStillCreatesWorktree(t *testing.T) {
	mainDir := initRepo(t, "main")
	cfg := config.Default()
	cfg.NoCode = true
	cfg.InstallCommands = []string{"false"}

	var out bytes.Buffer
	dir, err := New(cfg, mainDir, "myfeature", "", &out)
	if err != nil {
		t.Fatalf("New() error = %v, want nil (install failure should warn, not fail)", err)
	}

	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("worktree dir missing after install failure: %v", statErr)
	}
	if !bytes.Contains(out.Bytes(), []byte("WARNING install failed")) {
		t.Errorf("output missing install-failure warning:\n%s", out.String())
	}
}

// TestRemove_DetachedBranchGuardMatchesGitOutput exercises the exact guard
// expression Remove uses (target.Branch != git.DetachedBranch) against a
// real detached worktree, so it can't drift from what git.WorktreeList
// actually reports for the "(detached)" case. Remove itself isn't called
// directly here since it blocks on an interactive confirm prompt.
func TestRemove_DetachedBranchGuardMatchesGitOutput(t *testing.T) {
	mainDir := initRepo(t, "main")

	wtDir := mainDir + "-detached"
	cmd := exec.Command("git", "-C", mainDir, "worktree", "add", "--detach", "--", wtDir, "HEAD")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add --detach: %v\n%s", err, out)
	}

	entries, err := git.WorktreeList(mainDir)
	if err != nil {
		t.Fatal(err)
	}
	var target git.Worktree
	for _, e := range entries {
		if e.Path == wtDir {
			target = e
		}
	}
	if target.Path == "" {
		t.Fatalf("WorktreeList() = %+v, want an entry for %s", entries, wtDir)
	}

	if target.Branch != "" && target.Branch != git.DetachedBranch {
		t.Fatalf("Remove would attempt BranchDelete(%q) for a detached worktree", target.Branch)
	}
}
