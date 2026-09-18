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

func TestSuggest_SingleMarkerFile(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		wanted string
	}{
		{"go", "go.mod", "go mod download"},
		{"python", "requirements.txt", "pip install -r requirements.txt"},
		{"rust", "Cargo.toml", "cargo fetch"},
		{"ruby", "Gemfile", "bundle install"},
		{"php", "composer.json", "composer install"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}

			got := Suggest(dir)
			if len(got) != 1 || got[0] != tc.wanted {
				t.Errorf("Suggest() = %v, want [%q]", got, tc.wanted)
			}
		})
	}
}

func TestSuggest_DotNetDetectedByExtension(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{"csproj", "MyApp.csproj"},
		{"sln", "MyApp.sln"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}

			got := Suggest(dir)
			want := []string{"dotnet restore"}
			if len(got) != 1 || got[0] != want[0] {
				t.Errorf("Suggest() = %v, want %v", got, want)
			}
		})
	}
}

func TestSuggest_MavenPrefersWrapperOverGlobal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, want := Suggest(dir), []string{"mvn dependency:resolve"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("Suggest() without mvnw = %v, want %v", got, want)
	}

	if err := os.WriteFile(filepath.Join(dir, "mvnw"), []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got, want := Suggest(dir), []string{"./mvnw dependency:resolve"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("Suggest() with mvnw = %v, want %v", got, want)
	}
}

func TestSuggest_GradlePrefersWrapperOverGlobal(t *testing.T) {
	cases := []string{"build.gradle", "build.gradle.kts"}

	for _, buildFile := range cases {
		t.Run(buildFile, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, buildFile), []byte("plugins {}"), 0o644); err != nil {
				t.Fatal(err)
			}

			if got, want := Suggest(dir), []string{"gradle dependencies"}; len(got) != 1 || got[0] != want[0] {
				t.Errorf("Suggest() without gradlew = %v, want %v", got, want)
			}

			if err := os.WriteFile(filepath.Join(dir, "gradlew"), []byte("#!/bin/sh"), 0o755); err != nil {
				t.Fatal(err)
			}

			if got, want := Suggest(dir), []string{"./gradlew dependencies"}; len(got) != 1 || got[0] != want[0] {
				t.Errorf("Suggest() with gradlew = %v, want %v", got, want)
			}
		})
	}
}

func TestSuggest_JSLockfileWinsOverGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Suggest(dir); got != nil {
		t.Errorf("Suggest() = %v, want nil (Install already auto-detects the JS lockfile)", got)
	}
}

func TestSuggest_NothingDetected(t *testing.T) {
	dir := t.TempDir()
	if got := Suggest(dir); got != nil {
		t.Errorf("Suggest() = %v, want nil", got)
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
