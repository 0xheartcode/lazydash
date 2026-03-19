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
}

type Keybindings struct {
	OpenInBrowser string `yaml:"openInBrowser"`
	OpenInGhDash  string `yaml:"openInGhDash"`
	Refresh       string `yaml:"refresh"`
	Help          string `yaml:"help"`
	Quit          string `yaml:"quit"`
	PrevView      string `yaml:"prevView"`
	NextView      string `yaml:"nextView"`
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

	// Restore defaults for unset keybindings
	d := defaults()
	if cfg.Keybindings.OpenInBrowser == "" {
		cfg.Keybindings.OpenInBrowser = d.Keybindings.OpenInBrowser
	}
	if cfg.Keybindings.OpenInGhDash == "" {
		cfg.Keybindings.OpenInGhDash = d.Keybindings.OpenInGhDash
	}
	if cfg.Keybindings.Refresh == "" {
		cfg.Keybindings.Refresh = d.Keybindings.Refresh
	}
	if cfg.Keybindings.Help == "" {
		cfg.Keybindings.Help = d.Keybindings.Help
	}
	if cfg.Keybindings.Quit == "" {
		cfg.Keybindings.Quit = d.Keybindings.Quit
	}
	if cfg.Keybindings.PrevView == "" {
		cfg.Keybindings.PrevView = d.Keybindings.PrevView
	}
	if cfg.Keybindings.NextView == "" {
		cfg.Keybindings.NextView = d.Keybindings.NextView
	}
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
