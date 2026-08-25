package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Defaults    Defaults    `yaml:"defaults"`
	Keybindings Keybindings `yaml:"keybindings"`
}

type Defaults struct {
	Orgs                   []string `yaml:"orgs"`
	RefreshIntervalMinutes int      `yaml:"refreshIntervalMinutes"`
	IgnoreProjects         []string `yaml:"ignoreProjects"`
	IgnoreOrgs             []string `yaml:"ignoreOrgs"`
	OnlyOrgs               []string `yaml:"onlyOrgs"`
	OnlyProjects           []string `yaml:"onlyProjects"`
	// Sources selects which backends to enable: "github", "local". Empty means
	// auto-detect both (GitHub when gh is authenticated, local when the current
	// repository has git-native issues).
	Sources []string `yaml:"sources"`
}

type Keybindings struct {
	OpenInBrowser string `yaml:"openInBrowser"`
	OpenInGhDash  string `yaml:"openInGhDash"`
	Refresh       string `yaml:"refresh"`
	Help          string `yaml:"help"`
	Quit          string `yaml:"quit"`
	PrevView      string `yaml:"prevView"`
	NextView      string `yaml:"nextView"`
	Create        string `yaml:"create"`
	Comment       string `yaml:"comment"`
	ToggleState   string `yaml:"toggleState"`
	Move          string `yaml:"move"`
	Labels        string `yaml:"labels"`
	Assign        string `yaml:"assign"`
	Undo          string `yaml:"undo"`
}

func defaults() Config {
	return Config{
		Defaults: Defaults{
			Orgs:                   []string{},
			RefreshIntervalMinutes: 0,
		},
		Keybindings: Keybindings{
			OpenInBrowser: "o",
			OpenInGhDash:  "d",
			Refresh:       "r",
			Help:          "?",
			Quit:          "q",
			PrevView:      "[",
			NextView:      "]",
			Create:        "c",
			Comment:       "m",
			ToggleState:   "x",
			Move:          "M",
			Labels:        "L",
			Assign:        "a",
			Undo:          "u",
		},
	}
}

// Load reads config from the first location that exists, in priority order:
// 1. --config flag path
// 2. $LAZYDASH_CONFIG env var
// 3. .lazydash.yml in current dir
// 4. $XDG_CONFIG_HOME/lazydash/config.yml
func Load(flagPath string) (*Config, error) {
	cfg := defaults()

	path := resolvePath(flagPath)
	if path == "" {
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Restore defaults for any keybinding left unset in the config file.
	d := defaults().Keybindings
	kb := &cfg.Keybindings
	fill := func(dst *string, def string) {
		if *dst == "" {
			*dst = def
		}
	}
	fill(&kb.OpenInBrowser, d.OpenInBrowser)
	fill(&kb.OpenInGhDash, d.OpenInGhDash)
	fill(&kb.Refresh, d.Refresh)
	fill(&kb.Help, d.Help)
	fill(&kb.Quit, d.Quit)
	fill(&kb.PrevView, d.PrevView)
	fill(&kb.NextView, d.NextView)
	fill(&kb.Create, d.Create)
	fill(&kb.Comment, d.Comment)
	fill(&kb.ToggleState, d.ToggleState)
	fill(&kb.Move, d.Move)
	fill(&kb.Labels, d.Labels)
	fill(&kb.Assign, d.Assign)
	fill(&kb.Undo, d.Undo)
	return &cfg, nil
}

func resolvePath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	if env := os.Getenv("LAZYDASH_CONFIG"); env != "" {
		return env
	}
	if _, err := os.Stat(".lazydash.yml"); err == nil {
		return ".lazydash.yml"
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "lazydash", "config.yml")
}
