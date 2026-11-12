// Styled terminal output; colors are stripped when not a TTY or NO_COLOR is set.
package main

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	red    = lipgloss.NewStyle().Foreground(lipgloss.Red)
	green  = lipgloss.NewStyle().Foreground(lipgloss.Green)
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	bold   = lipgloss.NewStyle().Bold(true)
)

// errorf prints a red error line; status output is best-effort.
func errorf(w io.Writer, format string, a ...any) {
	_, _ = lipgloss.Fprintln(w, red.Render("✗ "+fmt.Sprintf(format, a...)))
}

// successf prints a green success line.
func successf(w io.Writer, format string, a ...any) {
	_, _ = lipgloss.Fprintln(w, green.Render("✓ "+fmt.Sprintf(format, a...)))
}

// warnf prints a yellow warning line.
func warnf(w io.Writer, format string, a ...any) {
	_, _ = lipgloss.Fprintln(w, yellow.Render("! "+fmt.Sprintf(format, a...)))
}

// printDiff prints a unified diff with colored headers, hunks and changes.
func printDiff(w io.Writer, diff string) error {
	var b strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		text := strings.TrimSuffix(line, "\n")
		switch {
		case text == "":
		case strings.HasPrefix(text, "---"), strings.HasPrefix(text, "+++"):
			text = bold.Render(text)
		case strings.HasPrefix(text, "@@"):
			text = cyan.Render(text)
		case strings.HasPrefix(text, "-"):
			text = red.Render(text)
		case strings.HasPrefix(text, "+"):
			text = green.Render(text)
		}
		b.WriteString(text)
		if strings.HasSuffix(line, "\n") {
			b.WriteByte('\n')
		}
	}
	_, err := lipgloss.Fprint(w, b.String())
	return err
}
