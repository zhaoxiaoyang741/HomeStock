package reply

import "strings"

// emoji constants for consistent message prefixes.
const (
	emojiSuccess = "✅"
	emojiError   = "❌"
	emojiWarning = "⚠️"
	emojiInfo    = "ℹ️"
	emojiList    = "📦"
	emojiEmpty   = "📭"
	emojiRefresh = "🔄"
)

// ForChannel creates a RenderContext from a channel name.
func ForChannel(channelName string) RenderContext {
	switch channelName {
	case "feishu":
		return RenderContext{Fancy: true}
	default:
		return RenderContext{Fancy: false}
	}
}

// --- Short message builders ---

func Success(_ RenderContext, msg string) string {
	return emojiSuccess + " " + msg
}

func Error(_ RenderContext, msg string) string {
	return emojiError + " " + msg
}

func Warning(_ RenderContext, msg string) string {
	return emojiWarning + " " + msg
}

func Info(_ RenderContext, msg string) string {
	return emojiInfo + " " + msg
}

func Empty(_ RenderContext, msg string) string {
	return emojiEmpty + " " + msg
}

// --- Internal table helper ---

// markdownTable renders a markdown pipe table.
func markdownTable(headers []string, rows [][]string) string {
	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")

	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("| " + strings.Join(sep, " | ") + " |\n")

	for _, row := range rows {
		escaped := make([]string, len(row))
		for i, cell := range row {
			escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
		}
		b.WriteString("| " + strings.Join(escaped, " | ") + " |\n")
	}
	return b.String()
}
