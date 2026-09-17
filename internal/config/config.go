// Package config resolves wt's configuration by layering defaults, the
// global config, the per-repo config, and one-off env var overrides.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds wt's fully resolved settings.
//
// WorktreeDir and BranchPattern are templates: "{repo}" and "{name}" are
// substituted by the caller once the repo name / worktree name are known.
type Config struct {
	WorktreeDir    string   `toml:"worktree_dir"`
	BranchPattern  string   `toml:"branch_pattern"`
	Editor         string   `toml:"editor"`
	EnvFiles       []string `toml:"env_files"`
	InstallCommand string   `toml:"install_command"`

	// NoInstall / NoCode come only from env vars (WT_NO_INSTALL /
	// WT_NO_CODE), never from a TOML file — they're one-shot run
	// overrides, not persistent settings.
	NoInstall bool
	NoCode    bool
}

// Default returns wt's built-in defaults, matching the original bash
// script's behavior.
func Default() Config {
	return Config{
		WorktreeDir:    "../{repo}-wt",
		BranchPattern:  "feature/{name}",
		Editor:         "code",
		EnvFiles:       []string{".env", ".env.local"},
		InstallCommand: "",
	}
}

// Load resolves the final config for a repo whose main checkout root is
// mainDir: defaults, then ~/.config/wt/config.toml, then <mainDir>/.wt.toml,
// then env var overrides. Missing files are not an error.
func Load(mainDir string) (Config, error) {
	cfg := Default()

	if path, err := globalConfigPath(); err == nil {
		if err := mergeFile(&cfg, path); err != nil {
			return Config{}, err
		}
	}

	if err := mergeFile(&cfg, filepath.Join(mainDir, ".wt.toml")); err != nil {
		return Config{}, err
	}

	applyEnv(&cfg)

	return cfg, nil
}

// mergeFile decodes path's TOML directly into cfg. toml.DecodeFile only
// sets the fields present in the file, so fields cfg already carries from
// an earlier, lower-priority layer are left untouched — this is what makes
// sequential decoding double as a merge.
func mergeFile(cfg *Config, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	_, err := toml.DecodeFile(path, cfg)
	return err
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "wt", "config.toml"), nil
}

func applyEnv(cfg *Config) {
	if v, ok := os.LookupEnv("WT_ENV_FILES"); ok {
		cfg.EnvFiles = strings.Fields(v)
	}
	if v := os.Getenv("WT_NO_INSTALL"); v != "" {
		cfg.NoInstall = true
	}
	if v := os.Getenv("WT_NO_CODE"); v != "" {
		cfg.NoCode = true
	}
}
