package tui

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	var lineLen int
	for i, word := range strings.Fields(text) {
		if lineLen > 0 && lineLen+1+len(word) > width {
			result.WriteRune('\n')
			lineLen = 0
		} else if i > 0 {
			result.WriteRune(' ')
			lineLen++
		}
		result.WriteString(word)
		lineLen += len(word)
	}
	return result.String()
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
