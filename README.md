# lazydash

A keyboard-driven terminal UI for GitHub Projects v2.

```
┌─ lazydash ──────────────────────────────────────────────────────┐
│ Projects (3)         Todo      In Progress   Done               │
│ ─────────────────    ───────   ───────────   ────   ──────────  │
│ > My Project         > Fix bug  Review PR    Deploy │ Fix bug   │
│   Team Q1              Add docs              v2     │ ───────── │
│   Roadmap              Update UI                    │ Issue #42 │
│                                                     │ Repo: org │
│                                                     │ Status:   │
│                                                     │   Todo    │
├─────────────────────────────────────────────────────────────────┤
│ j/k move  h/l columns  tab panes  o browser  ? help  q quit     │
└─────────────────────────────────────────────────────────────────┘
```

## Install

Requires [gh CLI](https://cli.github.com/) authenticated (`gh auth login`).

```bash
go install github.com/0xheartcode/lazydash@latest
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
| `o` | Open in browser |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

## Config

Config is loaded from the first location found:

1. `--config` flag
2. `$LAZYDASH_CONFIG` env var
3. `.lazydash.yml` in current directory
4. `$XDG_CONFIG_HOME/lazydash/config.yml` (default: `~/.config/lazydash/config.yml`)

```yaml
defaults:
  orgs: []                       # additional GitHub orgs to show projects from
  refreshIntervalMinutes: 5

keybindings:
  openInBrowser: o
  refresh: r
  help: "?"
  quit: q
```

## License

MIT
