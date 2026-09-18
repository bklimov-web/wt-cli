package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall_OverrideCommandsRunInOrder(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker.txt")

	err := Install(dir, []string{
		"echo one >> marker.txt",
		"echo two >> marker.txt",
	})
	if err != nil {
		t.Fatalf("Install() error = %v, want nil", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker file: %v", err)
	}
	want := "one\ntwo\n"
	if string(got) != want {
		t.Errorf("marker file = %q, want %q", got, want)
	}
}

func TestInstall_OverrideCommandsStopAtFirstFailure(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker.txt")

	err := Install(dir, []string{
		"false",
		"echo should-not-run >> marker.txt",
	})
	if err == nil {
		t.Fatal("Install() error = nil, want error from failing command")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Error("marker file exists, want second command to have been skipped")
	}
}

func TestInstall_NoOverrideNoLockfile(t *testing.T) {
	dir := t.TempDir()

	if err := Install(dir, nil); err != nil {
		t.Errorf("Install() error = %v, want nil (no-op)", err)
	}
}

func TestInstall_OverrideTakesPriorityOverLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "marker.txt")

	err := Install(dir, []string{"echo override >> marker.txt"})
	if err != nil {
		t.Fatalf("Install() error = %v, want nil", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker file: %v", err)
	}
	if string(got) != "override\n" {
		t.Errorf("marker file = %q, want %q", got, "override\n")
	}
}
