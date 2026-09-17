package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// writeFile is a small test helper: create parent dirs and write content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setHome points $HOME at a fresh temp dir so tests don't touch the real
// ~/.config/wt/config.toml, and don't leak state between tests.
func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		global  string // global config.toml content; "" = file not written
		perRepo string // .wt.toml content; "" = file not written
		env     map[string]string
		want    func(d Config) Config // mutate a copy of Default() into the expectation
	}{
		{
			name: "no files, no env: pure defaults",
			want: func(d Config) Config { return d },
		},
		{
			name:   "global overrides one field, rest stays default",
			global: `editor = "vim"`,
			want: func(d Config) Config {
				d.Editor = "vim"
				return d
			},
		},
		{
			name:    "per-repo wins over global for the field both set",
			global:  "editor = \"vim\"\nbranch_pattern = \"feature/{name}\"",
			perRepo: `editor = "subl"`,
			want: func(d Config) Config {
				d.Editor = "subl"
				// branch_pattern only set in global, so global's value
				// (which happens to equal the default) still applies.
				d.BranchPattern = "feature/{name}"
				return d
			},
		},
		{
			name:    "env vars override files and defaults",
			global:  `env_files = [".env.global"]`,
			perRepo: `env_files = [".env.repo"]`,
			env: map[string]string{
				"WT_ENV_FILES":  ".env.local .env.secrets",
				"WT_NO_INSTALL": "1",
				"WT_NO_CODE":    "1",
			},
			want: func(d Config) Config {
				d.EnvFiles = []string{".env.local", ".env.secrets"}
				d.NoInstall = true
				d.NoCode = true
				return d
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := setHome(t)
			mainDir := t.TempDir()

			if tc.global != "" {
				writeFile(t, filepath.Join(home, ".config", "wt", "config.toml"), tc.global)
			}
			if tc.perRepo != "" {
				writeFile(t, filepath.Join(mainDir, ".wt.toml"), tc.perRepo)
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := Load(mainDir)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			want := tc.want(Default())
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Load() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
