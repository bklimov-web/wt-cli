package worktree

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a bare-minimum git repo (no commits, no remote) —
// Init only needs `git status --ignored` to work, which doesn't require
// either.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

func TestInit_DetectsGoModAndIgnoredEnvFile(t *testing.T) {
	dir := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Init(dir, &out); err != nil {
		t.Fatalf("Init() error = %v\noutput:\n%s", err, out.String())
	}

	data, err := os.ReadFile(filepath.Join(dir, ".wt.toml"))
	if err != nil {
		t.Fatalf("reading .wt.toml: %v", err)
	}
	if !bytes.Contains(data, []byte(`install_commands = ["go mod download"]`)) {
		t.Errorf(".wt.toml missing install_commands:\n%s", data)
	}
	if !bytes.Contains(data, []byte(`env_files = [".env"]`)) {
		t.Errorf(".wt.toml missing env_files:\n%s", data)
	}
}

func TestInit_IgnoresUnrelatedIgnoredFiles(t *testing.T) {
	dir := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build", "out.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Init(dir, &out); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".wt.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("env_files")) {
		t.Errorf(".wt.toml should not list unrelated ignored files:\n%s", data)
	}
}

func TestInit_RefusesToOverwriteExisting(t *testing.T) {
	dir := initGitRepo(t)
	existing := filepath.Join(dir, ".wt.toml")
	if err := os.WriteFile(existing, []byte("editor = \"vim\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := Init(dir, &out)
	if err == nil {
		t.Fatal("Init() error = nil, want error (should not overwrite)")
	}

	data, readErr := os.ReadFile(existing)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "editor = \"vim\"\n" {
		t.Errorf(".wt.toml was modified, want untouched: %q", data)
	}
}

func TestInit_NothingDetectedIsAnError(t *testing.T) {
	dir := initGitRepo(t)

	var out bytes.Buffer
	if err := Init(dir, &out); err == nil {
		t.Fatal("Init() error = nil, want error (nothing to configure)")
	}

	if _, err := os.Stat(filepath.Join(dir, ".wt.toml")); !os.IsNotExist(err) {
		t.Error(".wt.toml should not have been written")
	}
}
