# lazydash

Humble keyboard-driven terminal UI for your issues — GitHub Projects **and** local git-native issues.
##### A browser (and editor) for your projects, in your terminal.

![demo](https://github.com/user-attachments/assets/0a1698ac-8a09-4426-a1f9-47f877aeffd2)


## Why

Are you too lazy to SWITCH from CLI to the browser like me ?
Use lazydash.

Your code travels with `git clone`; your GitHub issues don't. lazydash speaks to
both worlds from one keyboard-driven UI: your **GitHub Projects** boards, and
**local git-native issues** that live in the repo itself and work with no network
at all. Same board, same table, same keys — online or offline.

Also thinking of renaming it lazyview, since with alias lazydash = ld, which may already be taken by lazydocker.

Open an issue if you have a better name.

I will be working on this on and off.

## Backends

lazydash reads and writes through pluggable backends and shows them side by side
in one project list (tagged `gh` / `local` when you have both):

| Backend | Where issues live | Needs network | Manage (create/comment/close/move) |
|---|---|---|---|
| **GitHub Projects** | github.com | yes | comment, close/reopen, move card (needs `gh` CLI) |
| **Local** ([git-native-issue](https://github.com/remenoscodes/git-native-issue)) | this repo, under `refs/issues/` | no | full: create, comment, close/reopen, move, label, assign, undo |

Backends are auto-detected: GitHub when `gh` is authenticated, local when the
current repository has any `refs/issues/*`. Restrict them with the `sources`
config key.

## Install

For GitHub, [gh CLI](https://cli.github.com/) authenticated (`gh auth login`) —
also used to write (comment / close / move). For local issues you only need
`git`; nothing else, no network.

```bash
go install github.com/0xheartcode/lazydash@latest
```

Make sure `$HOME/go/bin` is in your PATH:

```bash
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc  # or ~/.zshrc
```

Or build from source:

```bash
git clone https://github.com/0xheartcode/lazydash
cd lazydash
go build -o lazydash .
```

## Usage

```bash
lazydash                                    # browse GitHub Projects + this repo's local issues
lazydash --config ~/.config/lazydash/config.yml
```

Run it **inside a repository** to work with that repo's local issues offline. The
local backend reads issues straight from git plumbing, and authors writes as
git commits under `refs/issues/`, so it interoperates with the `git issue` CLI
and needs no extra tooling.

## Keybindings

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `h` / `l` | Move left / right (board columns) |
| `tab` / `shift+tab` | Next / previous pane |
| `enter` | Load project · open card details |
| `esc` | Close details / cancel a modal |
| `[` / `]` | Cycle project views (Board → Table → Roadmap) |
| `o` | Open in browser |
| `d` | Open in gh-dash |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

Write actions (shown only for writable sources, gated per backend capability):

| Key | Action |
|---|---|
| `c` | New issue |
| `m` | Comment on the selected card |
| `x` | Close / reopen (with confirmation) |
| `M` | Move to another column (single-select field) |
| `L` / `a` | Set labels / assignee (local) |
| `u` | Undo last change (local issues) |

## Features

- **Two backends, one UI.** Browse and edit GitHub Projects and local git-native
  issues from the same board/table/roadmap views, switchable with `[` / `]`.
- **Fully offline for local issues.** Create, comment, close, move, label, assign
  and undo — all authored as git commits under `refs/issues/`, no network, no
  external binary.
- **Manage GitHub too.** Comment, close/reopen and move cards between columns via
  the authenticated `gh` CLI.
- **Offline cache.** The last synced GitHub board is cached, so it stays
  browsable (read-only) when you're offline.
- **Live GitHub views.** GitHub boards/tables/roadmaps are fetched live from the
  Projects API, so the TUI matches the browser.

## Config

Config is loaded from the first location found:

1. `--config` flag
2. `$LAZYDASH_CONFIG` env var
3. `.lazydash.yml` in current directory
4. `$XDG_CONFIG_HOME/lazydash/config.yml` (default: `~/.config/lazydash/config.yml`)

```yaml
defaults:
  sources: []                    # backends to enable: ["github", "local"]. empty = auto-detect both
  orgs: []                       # additional GitHub orgs to show projects from
  ignoreProjects: []             # projects to hide e.g. ["owner/name", "name"]
  ignoreOrgs: []                 # orgs to hide e.g. ["myorg"]
  onlyProjects: []               # if set, show only these projects
  onlyOrgs: []                   # if set, use only these orgs
  refreshIntervalMinutes: 5

keybindings:
  openInBrowser: o
  openInGhDash: d
  refresh: r
  help: "?"
  quit: q
  prevView: "["
  nextView: "]"
  # write actions
  create: c
  comment: m
  toggleState: x
  move: M
  labels: L
  assign: a
  undo: u
```

## Contribute

Open to contributions and PRs!

## License

MIT
