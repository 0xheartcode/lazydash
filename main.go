// Command lazydash is a keyboard-driven terminal UI for browsing and managing
// issues from one place: GitHub Projects boards and local git-native issues
// (issues stored in the repository under refs/issues/).
//
// Run it inside a repository to work with local issues fully offline — create,
// comment, close, move and undo — or anywhere to browse and edit your GitHub
// Projects. It auto-detects which backends are available. See the README for
// keybindings and configuration.
package main

import "github.com/0xheartcode/lazydash/cmd"

func main() {
	cmd.Execute()
}
