package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// InventoryTools holds service dependencies for inventory-related tool handlers.
type InventoryTools struct {
	InventorySvc *service.InventoryService
	MaterialSvc  *service.MaterialService
}

func (it *InventoryTools) InboundStock(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	spec, _ := args["spec"].(string)
	categoryID, _ := args["category_id"].(string)
	quantity, _ := args["quantity"].(float64)
	unit, _ := args["unit"].(string)
	location, _ := args["location"].(string)
	notes, _ := args["notes"].(string)

	if name == "" {
		return "", fmt.Errorf("inbound_stock: name is required")
	}
	if quantity <= 0 {
		return "", fmt.Errorf("inbound_stock: quantity must be positive")
	}

	var expireAt *time.Time
	if expireStr, ok := args["expire_at"].(string); ok && expireStr != "" {
		t, err := time.Parse("2006-01-02", expireStr)
		if err == nil {
			expireAt = &t
		}
	}

	lot, err := it.InventorySvc.Inbound(ctx, actor, service.InboundInput{
		TenantID:   actor.TenantID,
		Name:       name,
		Spec:       spec,
		CategoryID: categoryID,
		Quantity:   quantity,
		Unit:       unit,
		Location:   location,
		ExpireAt:   expireAt,
		Notes:      notes,
	})
	if err != nil {
		return "", fmt.Errorf("入库失败: %w", err)
	}

	result := fmt.Sprintf("✅ 入库成功！\n物品：%s\n数量：%.2f %s\n位置：%s\n批次：%s",
		lot.Material.Name, lot.QuantityOnHand, lot.Unit, lot.Location, lot.ID)
	if lot.ExpireAt != nil {
		result += fmt.Sprintf("\n过期日期：%s", lot.ExpireAt.Format("2006-01-02"))
	}
	return result, nil
}

func (it *InventoryTools) ConsumeMaterial(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
	materialID, _ := args["material_id"].(string)
	quantity, _ := args["quantity"].(float64)
	reason, _ := args["reason"].(string)

	if materialID == "" {
		return "", fmt.Errorf("consume_material: material_id is required")
	}
	if quantity <= 0 {
		return "", fmt.Errorf("consume_material: quantity must be positive")
	}

	results, err := it.InventorySvc.Consume(ctx, actor, materialID, actor.TenantID, quantity, reason)
	if err != nil {
		return "", fmt.Errorf("消耗失败: %w", err)
	}

	total := 0.0
	detail := ""
	for _, r := range results {
		total += r.ConsumedQuantity
		detail += fmt.Sprintf("\n- 批次 %s: 消耗 %.2f，剩余 %.2f", r.LotID, r.ConsumedQuantity, r.RemainingQuantity)
	}
	return fmt.Sprintf("✅ 消耗成功！共消耗 %.2f%s", total, detail), nil
}

func (it *InventoryTools) QueryInventory(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
	keyword, _ := args["keyword"].(string)
	categoryID, _ := args["category_id"].(string)
	location, _ := args["location"].(string)

	filter := repository.StockLotFilter{
		TenantID: actor.TenantID,
		Keyword:  keyword,
		Location: location,
	}
	if categoryID != "" {
		filter.CategoryID = categoryID
	}

	lots, err := it.InventorySvc.ListLots(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("查询失败: %w", err)
	}

	if len(lots) == 0 {
		return "📭 没有找到匹配的库存记录。", nil
	}

	type lotInfo struct {
		Name     string
		Qty      float64
		Unit     string
		Location string
		ExpireAt *time.Time
	}
	items := make([]lotInfo, 0, len(lots))
	for _, l := range lots {
		name := l.Material.Name
		if l.Material.Spec != "" {
			name += " (" + l.Material.Spec + ")"
		}
		items = append(items, lotInfo{
			Name:     name,
			Qty:      l.QuantityOnHand,
			Unit:     l.Unit,
			Location: l.Location,
			ExpireAt: l.ExpireAt,
		})
	}

	result := fmt.Sprintf("📦 共找到 %d 条记录：\n", len(items))
	for _, item := range items {
		line := fmt.Sprintf("\n- %s: %.2f %s [%s]", item.Name, item.Qty, item.Unit, item.Location)
		if item.ExpireAt != nil {
			line += fmt.Sprintf(" (过期: %s)", item.ExpireAt.Format("2006-01-02"))
		}
		result += line
	}
	return result, nil
}

func (it *InventoryTools) UpdateStockLot(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
	lotID, _ := args["lot_id"].(string)
	if lotID == "" {
		return "", fmt.Errorf("update_stock_lot: lot_id is required")
	}

	input := service.UpdateLotInput{}
	if expireStr, ok := args["expire_at"].(string); ok && expireStr != "" {
		t, err := time.Parse("2006-01-02", expireStr)
		if err == nil {
			input.ExpireAt = &t
		}
	}
	if loc, ok := args["location"].(string); ok {
		input.Location = &loc
	}
	if notes, ok := args["notes"].(string); ok {
		input.Notes = &notes
	}

	if qty, ok := args["quantity"].(float64); ok && qty >= 0 {
		updated, err := it.InventorySvc.Adjust(ctx, actor, lotID, actor.TenantID, qty, "调整", "")
		if err != nil {
			return "", fmt.Errorf("更新失败: %w", err)
		}
		if input.ExpireAt != nil || input.Location != nil || input.Notes != nil {
			updated, err = it.InventorySvc.UpdateLot(ctx, actor, lotID, actor.TenantID, input)
			if err != nil {
				return "", fmt.Errorf("更新信息失败: %w", err)
			}
		}
		return fmt.Sprintf("✅ 批次 %s 已更新！当前数量: %.2f %s", lotID, updated.QuantityOnHand, updated.Unit), nil
	}

	updated, err := it.InventorySvc.UpdateLot(ctx, actor, lotID, actor.TenantID, input)
	if err != nil {
		return "", fmt.Errorf("更新失败: %w", err)
	}
	return fmt.Sprintf("✅ 批次 %s 信息已更新！当前数量: %.2f %s", lotID, updated.QuantityOnHand, updated.Unit), nil
}

// InventoryToolDefinitions returns the LLM tool definitions for inventory operations.
func InventoryToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "inbound_stock",
				Description: "新增物品入库。支持通过名称查找或创建物料，记录批次、位置和过期信息。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":        map[string]any{"type": "string", "description": "物品名称，如'苹果'"},
						"spec":        map[string]any{"type": "string", "description": "规格，如'500g'"},
						"category_id": map[string]any{"type": "string", "description": "分类ID（可选）"},
						"quantity":    map[string]any{"type": "number", "description": "入库数量"},
						"unit":        map[string]any{"type": "string", "description": "单位，如'个'、'瓶'、'kg'"},
						"location":    map[string]any{"type": "string", "description": "存放位置，如'冰箱'、'橱柜'"},
						"expire_at":   map[string]any{"type": "string", "description": "过期日期，格式YYYY-MM-DD"},
						"notes":       map[string]any{"type": "string", "description": "备注"},
					},
					"required": []string{"name", "quantity"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "consume_material",
				Description: "消耗出库。按FIFO先进先出原则自动扣减库存批次。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"material_id": map[string]any{"type": "string", "description": "物料ID"},
						"quantity":    map[string]any{"type": "number", "description": "消耗数量"},
						"reason":      map[string]any{"type": "string", "description": "消耗原因，如'做饭用了'"},
					},
					"required": []string{"material_id", "quantity"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "query_inventory",
				Description: "查询库存。可按关键字、分类、位置筛选，返回库存批次列表。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword":     map[string]any{"type": "string", "description": "搜索关键字（物品名称）"},
						"category_id": map[string]any{"type": "string", "description": "按分类筛选"},
						"location":    map[string]any{"type": "string", "description": "按存放位置筛选"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "update_stock_lot",
				Description: "更新库存批次信息，包括修改数量、过期日期、存放位置和备注。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"lot_id":    map[string]any{"type": "string", "description": "批次ID"},
						"quantity":  map[string]any{"type": "number", "description": "调整后的数量（设为0即作废）"},
						"expire_at": map[string]any{"type": "string", "description": "过期日期，格式YYYY-MM-DD"},
						"location":  map[string]any{"type": "string", "description": "存放位置"},
						"notes":     map[string]any{"type": "string", "description": "备注"},
					},
					"required": []string{"lot_id"},
				},
			},
		},
	}
}

// RegisterInventoryTools registers all inventory tool handlers with the dispatcher.
func RegisterInventoryTools(d *Dispatcher, it *InventoryTools) {
	d.Register("inbound_stock", func(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
		return it.InboundStock(ctx, actor, args)
	})
	d.Register("consume_material", func(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
		return it.ConsumeMaterial(ctx, actor, args)
	})
	d.Register("query_inventory", func(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
		return it.QueryInventory(ctx, actor, args)
	})
	d.Register("update_stock_lot", func(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
		return it.UpdateStockLot(ctx, actor, args)
	})
}
