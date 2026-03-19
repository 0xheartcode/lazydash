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
}

type Keybindings struct {
	OpenInBrowser string `yaml:"openInBrowser"`
	Refresh       string `yaml:"refresh"`
	Help          string `yaml:"help"`
	Quit          string `yaml:"quit"`
}

func defaults() Config {
	return Config{
		Defaults: Defaults{
			Orgs:                   []string{},
			RefreshIntervalMinutes: 5,
		},
		Keybindings: Keybindings{
			OpenInBrowser: "o",
			Refresh:       "r",
			Help:          "?",
			Quit:          "q",
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
	if cfg.Keybindings.Refresh == "" {
		cfg.Keybindings.Refresh = d.Keybindings.Refresh
	}
	if cfg.Keybindings.Help == "" {
		cfg.Keybindings.Help = d.Keybindings.Help
	}
	if cfg.Keybindings.Quit == "" {
		cfg.Keybindings.Quit = d.Keybindings.Quit
	}
	if cfg.Defaults.RefreshIntervalMinutes == 0 {
		cfg.Defaults.RefreshIntervalMinutes = d.Defaults.RefreshIntervalMinutes
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
