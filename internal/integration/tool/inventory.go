package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/reply"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
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
	materialID, _ := args["material_id"].(string)

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

	svcLots, err := it.InventorySvc.Inbound(ctx, actor, service.InboundInput{
		TenantID:   actor.TenantID,
		Name:       name,
		Spec:       spec,
		CategoryID: categoryID,
		MaterialID: materialID,
		Quantity:   quantity,
		Unit:       unit,
		Location:   location,
		ExpireAt:   expireAt,
		Notes:      notes,
	})
	if err != nil {
		return "", fmt.Errorf("入库失败: %w", err)
	}
	if len(svcLots) == 0 {
		return "", fmt.Errorf("入库失败: 未创建任何批次")
	}

	// Use first lot for summary display
	first := svcLots[0]
	totalQty := 0.0
	for _, l := range svcLots {
		totalQty += l.QuantityOnHand
	}
	expireStr := ""
	if first.ExpireAt != nil {
		expireStr = first.ExpireAt.Format("2006-01-02")
	}
	rc := reply.ForChannel(actor.Channel)
	return reply.InboundSuccess(rc, reply.InventoryItemData{
		Name:     first.Material.Name,
		Spec:     first.Material.Spec,
		Quantity: totalQty,
		Unit:     first.Unit,
		Location: first.Location,
		ExpireAt: expireStr,
	}), nil
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
	details := make([]reply.ConsumeDetailData, 0, len(results))
	for _, r := range results {
		total += r.ConsumedQuantity
		details = append(details, reply.ConsumeDetailData{
			LotID:             r.LotID,
			ConsumedQuantity:  r.ConsumedQuantity,
			RemainingQuantity: r.RemainingQuantity,
		})
	}
	rc := reply.ForChannel(actor.Channel)
	return reply.ConsumeSuccess(rc, total, details), nil
}

func (it *InventoryTools) QueryInventory(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
	keyword, _ := args["keyword"].(string)
	categoryID, _ := args["category_id"].(string)
	location, _ := args["location"].(string)
	showZeroStock, _ := args["show_zero_stock"].(bool)

	filter := repository.StockLotFilter{
		TenantID:      actor.TenantID,
		Keyword:       keyword,
		Location:      location,
		ShowZeroStock: showZeroStock,
	}
	if categoryID != "" {
		filter.CategoryID = categoryID
	}

	lots, err := it.InventorySvc.ListLots(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("查询失败: %w", err)
	}

	if len(lots) == 0 {
		rc := reply.ForChannel(actor.Channel)
		return reply.Empty(rc, "没有找到匹配的库存记录。"), nil
	}

	items := make([]reply.InventoryItemData, 0, len(lots))
	for _, l := range lots {
		expireStr := ""
		if l.ExpireAt != nil {
			expireStr = l.ExpireAt.Format("2006-01-02")
		}
		items = append(items, reply.InventoryItemData{
			Name:     l.Material.Name,
			Spec:     l.Material.Spec,
			Quantity: l.QuantityOnHand,
			Unit:     l.Unit,
			Location: l.Location,
			ExpireAt: expireStr,
		})
	}

	rc := reply.ForChannel(actor.Channel)
	return reply.InventoryList(rc, items), nil
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
		rc := reply.ForChannel(actor.Channel)
		return reply.UpdateSuccess(rc, reply.UpdateResultData{
			LotID:    lotID,
			Quantity: updated.QuantityOnHand,
			Unit:     updated.Unit,
		}), nil
	}

	updated, err := it.InventorySvc.UpdateLot(ctx, actor, lotID, actor.TenantID, input)
	if err != nil {
		return "", fmt.Errorf("更新失败: %w", err)
	}
	rc := reply.ForChannel(actor.Channel)
	return reply.UpdateSuccess(rc, reply.UpdateResultData{
		LotID:    lotID,
		Quantity: updated.QuantityOnHand,
		Unit:     updated.Unit,
	}), nil
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
				Description: "查询库存。可按关键字、分类、位置筛选，返回库存批次列表。默认不返回零库存批次。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword":         map[string]any{"type": "string", "description": "搜索关键字（物品名称）"},
						"category_id":     map[string]any{"type": "string", "description": "按分类筛选"},
						"location":        map[string]any{"type": "string", "description": "按存放位置筛选"},
						"show_zero_stock": map[string]any{"type": "boolean", "description": "是否同时显示零库存批次，默认false"},
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
