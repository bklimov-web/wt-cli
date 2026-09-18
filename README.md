# wt

`wt` creates git worktrees with your local files (env files, editor,
installed deps) already in place, so switching to a new branch doesn't mean
re-copying `.env` and waiting for `npm install` by hand.

```
$ wt new my-feature
wt: fetching origin...
wt: copied .env
wt: running pnpm install
wt: feature/my-feature  ->  ../myrepo-wt/my-feature
```

## Install

### Homebrew (macOS/Linux, no Go required)

```sh
brew install bklimov-web/tap/wt
```

Installs as `wt`, updates with `brew upgrade wt`.

### Go

```sh
go install github.com/bklimov-web/wt-cli/cmd/wt@latest
```

Or clone and build:

```sh
git clone https://github.com/bklimov-web/wt-cli
cd wt-cli
go build -o wt ./cmd/wt
```

> **Note:** `wt` is a common name — it may collide with another binary on
> your `PATH` (some shells ship a `wt` alias for Windows Terminal, for
> example). If that happens, rename the built binary or alias it:
> `alias wt=/path/to/wt`.

## Commands

| Command | What it does |
| --- | --- |
| `wt new <name> [branch]` | Create a worktree branched off `origin/<default>`, copy env files, run the detected package manager's install, open it in your editor |
| `wt ls` | List worktrees (branch + path) |
| `wt rm [name]` | Remove a worktree and its branch, with a confirmation prompt. Interactive picker if `name` is omitted |
| `wt open [name]` | Open a worktree in your configured editor. Interactive picker if `name` is omitted |
| `wt path [name]` | Print a worktree's path. Interactive picker if `name` is omitted |

Run any command inside your main checkout (or one of its worktrees) — `wt`
resolves the main checkout automatically.

## Configuration

`wt` reads config from two optional TOML files, merged in this order
(later wins): defaults → `~/.config/wt/config.toml` → `.wt.toml` in your
repo root.

```toml
# ~/.config/wt/config.toml or <repo>/.wt.toml
worktree_dir     = "../{repo}-wt"     # {repo} is replaced with the repo's dir name
branch_pattern   = "feature/{name}"   # {name} is replaced with the worktree name
editor           = "code"             # command used to open a worktree
env_files        = [".env", ".env.local"]
install_commands = []                 # override auto-detected install step
```

Install-manager detection is by lockfile: `pnpm-lock.yaml` → `pnpm install`,
`yarn.lock` → `yarn install --frozen-lockfile`, `bun.lockb` → `bun install`,
`package-lock.json` → `npm ci`. No lockfile, no install.

`install_commands` overrides this detection with your own steps, run in
order through the shell (stopping at the first failure) — handy for
ecosystems with no single lockfile convention, e.g. Python:

```toml
install_commands = [
  "python -m venv .venv",
  ".venv/bin/pip install -r requirements.txt",
]
```

### Environment overrides

These apply on top of both config files, for one-off use:

| Variable | Effect |
| --- | --- |
| `WT_ENV_FILES` | Space-separated list, overrides `env_files` |
| `WT_NO_INSTALL` | Skip the install step |
| `WT_NO_CODE` | Skip opening the editor |

## Requirements

- Git with worktree support
- Go 1.25+ (only to build from source)

## Development

```sh
git clone https://github.com/bklimov-web/wt-cli
cd wt-cli
make setup   # git hooks (auto-gofmt on commit, build/vet/test on push) + deps
```

```sh
make build   # go build ./...
make test    # go test ./...
make vet     # go vet ./...
make lint    # golangci-lint run (needs golangci-lint installed)
make ci      # what CI runs: gofmt check + vet + build + test -race -shuffle=on
```

No `make`? Run what it wraps directly:

```sh
git config core.hooksPath .githooks   # setup
go mod download                       # setup
go build ./...                        # build
go test ./...                         # test
go vet ./...                          # vet
```

## License

[MIT](LICENSE)
