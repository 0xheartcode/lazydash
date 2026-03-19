package keys

import "github.com/0xheartcode/lazydash/internal/config"

// Bindings holds all key strings used by the TUI.
type Bindings struct {
	Up            string
	Down          string
	Left          string
	Right         string
	NextPane      string
	PrevPane      string
	OpenInBrowser string
	OpenInGhDash  string
	Refresh       string
	Help          string
	Quit          string
	Enter         string
	PrevView      string
	NextView      string
}

// FromConfig builds Bindings from config, filling in hard-coded navigation keys.
func FromConfig(cfg *config.Config) Bindings {
	return Bindings{
		Up:            "k",
		Down:          "j",
		Left:          "h",
		Right:         "l",
		NextPane:      "tab",
		PrevPane:      "shift+tab",
		Enter:         "enter",
		OpenInBrowser: cfg.Keybindings.OpenInBrowser,
		OpenInGhDash:  cfg.Keybindings.OpenInGhDash,
		Refresh:       cfg.Keybindings.Refresh,
		Help:          cfg.Keybindings.Help,
		Quit:          cfg.Keybindings.Quit,
		PrevView:      cfg.Keybindings.PrevView,
		NextView:      cfg.Keybindings.NextView,
	}
}
