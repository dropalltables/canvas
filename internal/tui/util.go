package tui

import (
	"os/exec"
	"regexp"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^\s)]+)(?:\s+"[^"]*")?\)`)

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return ansi.Wrap(text, width, "/?&=#:.")
}

func getNumberKey(msg tea.KeyMsg) int {
	if len(msg.String()) == 1 {
		c := msg.String()[0]
		if c >= '1' && c <= '9' {
			return int(c - '0')
		}
	}
	return 0
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
