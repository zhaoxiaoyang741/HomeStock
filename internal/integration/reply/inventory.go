package reply

import (
	"fmt"
	"strings"
)

// InboundSuccess formats an inbound stock success message.
func InboundSuccess(ctx RenderContext, data InventoryItemData) string {
	if ctx.Fancy {
		return inboundSuccessFancy(data)
	}
	return inboundSuccessPlain(data)
}

func inboundSuccessPlain(data InventoryItemData) string {
	var b strings.Builder
	b.WriteString(emojiSuccess + " 入库成功\n")
	b.WriteString(fmt.Sprintf("  名称：%s\n", nameWithSpec(data.Name, data.Spec)))
	b.WriteString(fmt.Sprintf("  数量：%.2f %s\n", data.Quantity, data.Unit))
	b.WriteString(fmt.Sprintf("  位置：%s\n", data.Location))
	if data.ExpireAt != "" {
		b.WriteString(fmt.Sprintf("  过期：%s", data.ExpireAt))
	}
	return b.String()
}

func inboundSuccessFancy(data InventoryItemData) string {
	var b strings.Builder
	b.WriteString(emojiSuccess + " 入库成功\n")
	rows := [][]string{
		{nameWithSpec(data.Name, data.Spec), fmt.Sprintf("%.2f %s", data.Quantity, data.Unit), data.Location, data.ExpireAt},
	}
	b.WriteString(markdownTable([]string{"物品", "数量", "位置", "过期"}, rows))
	return b.String()
}

// ConsumeSuccess formats a consumption success message.
func ConsumeSuccess(ctx RenderContext, total float64, details []ConsumeDetailData) string {
	if ctx.Fancy {
		return consumeSuccessFancy(total, details)
	}
	return consumeSuccessPlain(total, details)
}

func consumeSuccessPlain(total float64, details []ConsumeDetailData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s 消耗成功！共消耗 %.2f\n", emojiSuccess, total))
	for _, d := range details {
		b.WriteString(fmt.Sprintf("  - %s: 消耗 %.2f，剩余 %.2f\n", d.LotID, d.ConsumedQuantity, d.RemainingQuantity))
	}
	return b.String()
}

func consumeSuccessFancy(total float64, details []ConsumeDetailData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s 消耗成功！共消耗 %.2f\n", emojiSuccess, total))
	rows := make([][]string, 0, len(details))
	for _, d := range details {
		rows = append(rows, []string{
			d.LotID,
			fmt.Sprintf("%.2f", d.ConsumedQuantity),
			fmt.Sprintf("%.2f", d.RemainingQuantity),
		})
	}
	b.WriteString(markdownTable([]string{"批次", "消耗", "剩余"}, rows))
	return b.String()
}

// InventoryList formats an inventory query result.
func InventoryList(ctx RenderContext, items []InventoryItemData) string {
	if len(items) == 0 {
		return emojiEmpty + " 没有找到匹配的库存记录。"
	}
	if ctx.Fancy {
		return inventoryListFancy(items)
	}
	return inventoryListPlain(items)
}

func inventoryListPlain(items []InventoryItemData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s 共找到 %d 条记录：\n", emojiList, len(items)))
	for _, item := range items {
		b.WriteString(fmt.Sprintf("  %s: %.2f %s [%s]", nameWithSpec(item.Name, item.Spec), item.Quantity, item.Unit, item.Location))
		if item.ExpireAt != "" {
			b.WriteString(fmt.Sprintf(" (过期: %s)", item.ExpireAt))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func inventoryListFancy(items []InventoryItemData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s 共找到 %d 条记录\n", emojiList, len(items)))
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		expireAt := item.ExpireAt
		if expireAt == "" {
			expireAt = "无"
		}
		rows = append(rows, []string{
			nameWithSpec(item.Name, item.Spec),
			fmt.Sprintf("%.2f %s", item.Quantity, item.Unit),
			item.Location,
			expireAt,
		})
	}
	b.WriteString(markdownTable([]string{"物品", "数量", "位置", "过期"}, rows))
	return b.String()
}

// UpdateSuccess formats a lot update success message.
func UpdateSuccess(ctx RenderContext, data UpdateResultData) string {
	return fmt.Sprintf("%s 批次 %s 已更新！当前数量: %.2f %s", emojiSuccess, data.LotID, data.Quantity, data.Unit)
}

// nameWithSpec joins name and spec if spec is non-empty.
func nameWithSpec(name, spec string) string {
	if spec != "" {
		return name + " (" + spec + ")"
	}
	return name
}
