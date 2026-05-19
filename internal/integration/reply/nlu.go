package reply

import (
	"fmt"
	"strings"
)

// ConfirmDisambiguation formats a name disambiguation message.
func ConfirmDisambiguation(name string, candidates []ConfirmCandidateData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("找到多个「%s」：\n", name))
	labels := []string{"A", "B", "C", "D", "E"}
	for i, c := range candidates {
		if i >= len(labels) {
			break
		}
		b.WriteString(fmt.Sprintf("  %s. %s", labels[i], c.Name))
		if c.Spec != "" {
			b.WriteString(fmt.Sprintf(" (%s)", c.Spec))
		}
		b.WriteString(fmt.Sprintf(" — 单位: %s\n", c.Unit))
	}
	if len(candidates) > len(labels) {
		b.WriteString(fmt.Sprintf("  ...以及其他 %d 个\n", len(candidates)-len(labels)))
	}
	b.WriteString("请回复选项字母（如 A、B、C），或输入更精确的名称。")
	return b.String()
}
