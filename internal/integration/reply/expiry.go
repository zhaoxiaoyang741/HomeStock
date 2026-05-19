package reply

import (
	"fmt"
	"strings"
)

// ExpiryWarning formats an expiry notification message.
func ExpiryWarning(ctx RenderContext, items []ExpiryItemData) string {
	if ctx.Fancy {
		return expiryWarningFancy(items)
	}
	return expiryWarningPlain(items)
}

func expiryWarningPlain(items []ExpiryItemData) string {
	var b strings.Builder
	b.WriteString("以下食材即将过期：\n")
	for _, item := range items {
		nameStr := nameWithSpec(item.Name, item.Spec)
		b.WriteString(fmt.Sprintf("\n  %s\n", nameStr))
		b.WriteString(fmt.Sprintf("  数量: %.1f %s\n", item.Quantity, item.Unit))
		b.WriteString(fmt.Sprintf("  过期: %s\n", item.ExpireAt))
		b.WriteString(fmt.Sprintf("  存放: %s", item.Location))
	}
	return b.String()
}

func expiryWarningFancy(items []ExpiryItemData) string {
	var b strings.Builder
	b.WriteString(emojiWarning + " 以下食材即将过期\n")
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			nameWithSpec(item.Name, item.Spec),
			fmt.Sprintf("%.1f %s", item.Quantity, item.Unit),
			item.ExpireAt,
			item.Location,
		})
	}
	b.WriteString(markdownTable([]string{"物品", "数量", "过期", "存放"}, rows))
	return b.String()
}
