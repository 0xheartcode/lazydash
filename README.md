# lazydash

Humble keyboard-driven terminal UI for GitHub Projects.
##### A browser for your projects, in your terminal.

![demo](https://github.com/user-attachments/assets/0a1698ac-8a09-4426-a1f9-47f877aeffd2)


## Why

Are you too lazy to SWITCH from CLI to the browser like me ? 
Use lazydash. 

Also thinking of renaming it lazyview, since with alias lazydash = ld, which may already be taken by lazydocker.

Open an issue if you have a better name.

I will be working on this on and off.

## Install

Requires [gh CLI](https://cli.github.com/) authenticated (`gh auth login`).

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
lazydash
lazydash --config ~/.config/lazydash/config.yml
```

## Keybindings

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `h` / `l` | Move left / right (board columns) |
| `tab` | Next pane |
| `shift+tab` | Previous pane |
| `enter` | Select |
| `[` / `]` | Cycle project views (Board → Table → Roadmap) |
| `o` | Open in browser |
| `d` | Open in gh-dash |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

## Features

lazydash mirrors your GitHub Project views — Board, Table, and Roadmap — and lets you switch between them with `[` / `]`. Views are fetched live from the GitHub Projects API so what you see in the TUI matches what you see in the browser.

## Config

Config is loaded from the first location found:

1. `--config` flag
2. `$LAZYDASH_CONFIG` env var
3. `.lazydash.yml` in current directory
4. `$XDG_CONFIG_HOME/lazydash/config.yml` (default: `~/.config/lazydash/config.yml`)

```yaml
defaults:
  orgs: []                       # additional GitHub orgs to show projects from
  ignoreProjects: []             # projects to hide e.g. ["owner/name", "name"]
  ignoreOrgs: []                 # orgs to hide e.g. ["myorg"]
  refreshIntervalMinutes: 5

keybindings:
  openInBrowser: o
  openInGhDash: d
  refresh: r
  help: "?"
  quit: q
```
## Contribute

Open to contributions and PRs!

## License

MIT
