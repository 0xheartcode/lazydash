package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Base colors
	ColorPrimary   = lipgloss.Color("99")  // purple
	ColorSecondary = lipgloss.Color("212") // pink
	ColorMuted     = lipgloss.Color("241")
	ColorSuccess   = lipgloss.Color("76")
	ColorWarning   = lipgloss.Color("214")
	ColorDanger    = lipgloss.Color("196")
	ColorText      = lipgloss.Color("252")
	ColorSubtext   = lipgloss.Color("244")
	ColorBg        = lipgloss.Color("235")
	ColorBorder    = lipgloss.Color("238")
	ColorActive    = lipgloss.Color("99")

	// Pane borders
	InactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorActive)

	// Typography
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	Subtitle = lipgloss.NewStyle().
			Foreground(ColorSubtext)

	Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Cards
	CardCursor = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	CardTitle = lipgloss.NewStyle().
			Foreground(ColorText)

	CardSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Column headers
	ColumnHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder)

	// Footer
	FooterKey = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	FooterDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)

	FooterBar = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder)

	// Status badges
	StatusOpen   = lipgloss.NewStyle().Foreground(ColorSuccess)
	StatusClosed = lipgloss.NewStyle().Foreground(ColorDanger)
	StatusMerged = lipgloss.NewStyle().Foreground(ColorPrimary)
	StatusDraft  = lipgloss.NewStyle().Foreground(ColorMuted)

	// Sidebar
	SidebarLabel = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Width(9)

	SidebarValue = lipgloss.NewStyle().
			Foreground(ColorText)

	// Help overlay
	HelpKey  = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(ColorMuted)
)

// StateColor returns the appropriate style for an issue/PR state string.
func StateColor(state string) lipgloss.Style {
	switch state {
	case "OPEN", "open":
		return StatusOpen
	case "CLOSED", "closed":
		return StatusClosed
	case "MERGED", "merged":
		return StatusMerged
	default:
		return StatusDraft
	}
}
