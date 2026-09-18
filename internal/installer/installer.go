// Package installer runs a worktree's install step: either a user-supplied
// list of commands, or an auto-detected JS package manager's install
// command based on lockfiles present. It also detects a suggested install
// command for other ecosystems (Go, Python, Java, Rust, Ruby, PHP, .NET)
// for `wt init` to write into .wt.toml — those don't get run
// automatically, since each build tool already fetches its own
// dependencies lazily.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// manager pairs a lockfile with the command that installs from it.
type manager struct {
	lockfile string
	command  []string
}

var managers = []manager{
	{"pnpm-lock.yaml", []string{"pnpm", "install"}},
	{"yarn.lock", []string{"yarn", "install", "--frozen-lockfile"}},
	{"bun.lockb", []string{"bun", "install"}},
	{"package-lock.json", []string{"npm", "ci"}},
}

// Install runs the worktree's install step in dir. If commands is
// non-empty, each entry is run in order through the shell (so entries can
// use shell syntax like "&&" or pipes), stopping at the first failure.
// Otherwise it falls back to detecting a JS package manager from dir's
// lockfile. It's a no-op (with a status line) if commands is empty and no
// known lockfile is present.
func Install(dir string, commands []string) error {
	if len(commands) > 0 {
		for _, c := range commands {
			fmt.Printf("wt: running %s\n", c)
			cmd := exec.Command("sh", "-c", c)
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
	}

	for _, m := range managers {
		if _, err := os.Stat(filepath.Join(dir, m.lockfile)); err != nil {
			continue
		}
		fmt.Printf("wt: running %s\n", strings.Join(m.command, " "))
		cmd := exec.Command(m.command[0], m.command[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Println("wt: no lockfile found, skipping install")
	return nil
}

// detector reports the shell command to suggest for dir's ecosystem, and
// whether it recognized one at all. Unlike the JS package managers above,
// each of these ecosystems has exactly one real install command, so
// there's nothing to disambiguate at install time; the only value in
// suggesting it is saving the user from typing the line into .wt.toml
// themselves.
type detector func(dir string) (command string, ok bool)

// marker builds a detector for an ecosystem with one command and one
// marker file.
func marker(file, command string) detector {
	return func(dir string) (string, bool) {
		return command, exists(filepath.Join(dir, file))
	}
}

// wrapper builds a detector for an ecosystem whose marker file may be
// paired with a version-pinned wrapper script (Maven's mvnw, Gradle's
// gradlew) that Spring-style projects commonly commit — when present, it
// should be invoked instead of a global install, since it's what pins the
// build tool's version for the whole team.
func wrapper(markerFile, wrapperFile, wrapperCommand, fallbackCommand string) detector {
	return func(dir string) (string, bool) {
		if !exists(filepath.Join(dir, markerFile)) {
			return "", false
		}
		if exists(filepath.Join(dir, wrapperFile)) {
			return wrapperCommand, true
		}
		return fallbackCommand, true
	}
}

// glob builds a detector matched by a filepath.Glob pattern, for
// ecosystems identified by a file extension rather than a fixed name
// (.NET's *.csproj / *.sln).
func glob(pattern, command string) detector {
	return func(dir string) (string, bool) {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		return command, len(matches) > 0
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var detectors = []detector{
	marker("go.mod", "go mod download"),
	marker("requirements.txt", "pip install -r requirements.txt"),
	wrapper("pom.xml", "mvnw", "./mvnw dependency:resolve", "mvn dependency:resolve"),
	wrapper("build.gradle", "gradlew", "./gradlew dependencies", "gradle dependencies"),
	wrapper("build.gradle.kts", "gradlew", "./gradlew dependencies", "gradle dependencies"),
	marker("Cargo.toml", "cargo fetch"),
	marker("Gemfile", "bundle install"),
	marker("composer.json", "composer install"),
	glob("*.csproj", "dotnet restore"),
	glob("*.sln", "dotnet restore"),
}

// Suggest returns install commands worth writing to .wt.toml for dir.
// Returns nil if a JS lockfile is present (Install already auto-detects
// those at runtime without any config needed) or if nothing was detected.
func Suggest(dir string) []string {
	for _, m := range managers {
		if exists(filepath.Join(dir, m.lockfile)) {
			return nil
		}
	}
	for _, d := range detectors {
		if cmd, ok := d(dir); ok {
			return []string{cmd}
		}
	}
	return nil
}
