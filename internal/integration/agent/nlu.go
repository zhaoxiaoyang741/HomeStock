package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// NluResult is the structured output from NLU extraction.
type NluResult struct {
	Intent  string            `json:"intent"`  // execute | clarify | confirm | reject | chitchat
	Actions []ExtractedAction `json:"actions"`
	RawText string            `json:"raw_text"`
}

// ExtractedAction is a single operation extracted from user input.
type ExtractedAction struct {
	Type       string         `json:"type"`       // inbound | consume | query | update | delete | undo
	Items      []ExtractedItem `json:"items"`
	Parameters map[string]any `json:"parameters"`
}

// ExtractedItem is a single item within an action (e.g. one product in multi-item inbound).
type ExtractedItem struct {
	Name               string             `json:"name"`
	Quantity           *float64           `json:"quantity"`
	Unit               string             `json:"unit"`
	Location           string             `json:"location"`
	ExpireAt           string             `json:"expire_at"`
	Spec               string             `json:"spec"`
	Confidence         map[string]float64 `json:"confidence"`
	ResolvedMaterialID string             `json:"-"` // set by name resolution, not from NLU
}

// NluEngine builds NLU prompts and parses structured responses.
// It does NOT make LLM calls — that is AgentLoop's responsibility.
type NluEngine struct {
	materialSvc *service.MaterialService
}

// NewNluEngine creates an NluEngine.
func NewNluEngine(materialSvc *service.MaterialService) *NluEngine {
	return &NluEngine{materialSvc: materialSvc}
}

var injectionPattern = regexp.MustCompile(`(?i)(ignore\s+(above|all|previous)|forget\s+(all|everything|previous)|你是\w+|you\s+are\s+\w+|忽略[以所].*指令|无视.*指令)`)

// SanitizeInput cleans user text: trim, cap 500 chars, strip injection patterns.
func (e *NluEngine) SanitizeInput(text string) string {
	s := strings.TrimSpace(text)
	if len(s) > 500 {
		s = s[:500]
	}
	if injectionPattern.MatchString(s) {
		return ""
	}
	return s
}

// PrefetchCatalog queries MaterialService.List with a keyword and returns formatted catalog text.
// If keyword is empty or no materials match, returns an empty catalog section.
func (e *NluEngine) PrefetchCatalog(ctx context.Context, keyword, tenantID string) string {
	if e.materialSvc == nil || keyword == "" {
		return ""
	}
	// Extract first non-trivial keyword (up to 10 chars, Chinese or alphanumeric)
	kw := extractKeyword(keyword)
	if kw == "" {
		return ""
	}

	summaries, err := e.materialSvc.List(ctx, repository.MaterialFilter{
		TenantID: tenantID,
		Keyword:  kw,
	})
	if err != nil || len(summaries) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n当前物料目录（只读参考）：\n")
	limit := 10
	if len(summaries) < limit {
		limit = len(summaries)
	}
	for i, s := range summaries[:limit] {
		b.WriteString(fmt.Sprintf("  %d. %s", i+1, s.Name))
		if s.Spec != "" {
			b.WriteString(fmt.Sprintf(" (%s)", s.Spec))
		}
		b.WriteString(fmt.Sprintf(" — 单位: %s, 库存: %.0f", s.DefaultUnit, s.TotalQuantity))
		if len(s.Locations) > 0 {
			b.WriteString(fmt.Sprintf(", 位置: %s", strings.Join(s.Locations, "/")))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// extractKeyword picks the first meaningful keyword from user input.
func extractKeyword(text string) string {
	fields := strings.Fields(text)
	for _, f := range fields {
		f = strings.Trim(f, "，。、！？；：,.!?;: \t\"'")
		if len([]rune(f)) >= 1 && len([]rune(f)) <= 15 {
			// Skip pure numbers and stop words
			if isPureNumber(f) || isStopWord(f) {
				continue
			}
			return f
		}
	}
	return ""
}

func isPureNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

var stopWords = map[string]bool{
	"买了": true, "买": true, "入库": true, "出库": true, "消耗": true,
	"查询": true, "查": true, "删除": true, "更新": true, "修改": true,
	"撤回": true, "撤销": true, "用了": true, "用": true, "有": true,
	"多少": true, "什么": true, "怎么": true, "如何": true, "的": true,
}

func isStopWord(s string) bool { return stopWords[s] }

// BuildNluSystemPrompt returns the full NLU extraction prompt with catalog and context injected.
func (e *NluEngine) BuildNluSystemPrompt(catalog, recentContext string) string {
	prompt := nluSystemPromptBase
	if catalog != "" {
		prompt += "\n\n" + catalog
	}
	if recentContext != "" {
		prompt += "\n最近对话上下文：\n" + recentContext
	}
	return prompt
}

// ParseResponse parses the raw LLM JSON response into NluResult.
// Attempts JSON parsing; if that fails, tries to extract JSON from markdown code blocks.
func (e *NluEngine) ParseResponse(raw string) (*NluResult, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Try direct JSON parse first
	var result NluResult
	if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
		return &result, nil
	}

	// Try extracting from markdown code block
	start := strings.Index(trimmed, "```")
	if start >= 0 {
		rest := trimmed[start+3:]
		if idx := strings.Index(rest, "\n"); idx >= 0 {
			rest = rest[idx+1:]
		}
		end := strings.Index(rest, "```")
		if end >= 0 {
			rest = rest[:end]
		}
		if err := json.Unmarshal([]byte(rest), &result); err == nil {
			return &result, nil
		}
	}

	// Try finding any JSON object in the response
	braceStart := strings.Index(trimmed, "{")
	if braceStart >= 0 {
		braceEnd := strings.LastIndex(trimmed, "}")
		if braceEnd > braceStart {
			jsonStr := trimmed[braceStart : braceEnd+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
				return &result, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot parse NLU response: %s", trimmed[:min(len(trimmed), 100)])
}

const nluSystemPromptBase = `你是一个家庭库存管理系统的自然语言理解引擎。
你的任务是从用户输入中提取库存管理意图和结构化数据。

## 意图分类
- execute: 明确的库存操作（入库、出库、查询、更新等）
- clarify: 用户表述模糊，无法确定具体操作
- chitchat: 问候、感谢等日常对话（非库存操作）
- reject: 用户输入与本系统无关或无法理解

## 操作类型
- inbound: 入库（买入、收到、补充、新增等）
- consume: 出库/消耗（使用、吃掉、用掉等）
- query: 查询库存（查、看、还有吗等）
- update: 更新批次信息
- delete: 删除/作废
- undo: 撤回上一步操作

## 推断规则（用户没说到的字段尽量推断）
1. 用户没提单位时，优先用物料目录中的默认单位
2. 用户没提数量时，普通物品默认推断为1份，除非用了"一些""一点"等模糊量词
3. 用户没提位置时，优先用物料目录中的最近存放位置
4. 有保质期数据的物料，从今天推算过期日期
5. 推断的值标注 confidence 0.6-0.8，不能推断的留null

## 输出格式
必须输出纯JSON，不要使用markdown代码块标记。
{
  "intent": "execute",
  "actions": [
    {
      "type": "inbound",
      "items": [
        {
          "name": "牛奶",
          "quantity": 2,
          "unit": "瓶",
          "location": "冰箱",
          "expire_at": "2026-05-21",
          "spec": "1L",
          "confidence": {"name": 1.0, "quantity": 1.0, "unit": 0.8, "location": 0.6, "expire_at": 0.6}
        }
      ],
      "parameters": {}
    }
  ],
  "raw_text": "买了2瓶牛奶"
}

## 规则
1. 一条输入可能包含多个操作，全部提取
2. 每个物品字段标注置信度（0-1）
3. 无法确定的字段设为null
4. 如果用户输入是"撤回"或"撤销"，设置intent为execute，type为undo
5. 数量词规范化："5斤"=quantity:5,unit:"斤"；"两瓶"=quantity:2
6. 根据物料目录中的名称、单位、位置信息辅助推断
7. 多物品时每个物品作为独立item输出`

// ResolveResult holds a fuzzy-matched candidate material.
type ResolveResult struct {
	MaterialID   string  `json:"material_id"`
	Name         string  `json:"name"`
	Spec         string  `json:"spec"`
	DefaultUnit  string  `json:"default_unit"`
	Score        float64 `json:"score"`
	IsExactMatch bool    `json:"is_exact_match"`
}

// NameResolver is the function signature for material name resolution.
type NameResolver func(ctx context.Context, name, tenantID string) ([]ResolveResult, error)

// DefaultNameResolver returns a NameResolver backed by MaterialService.List.
func DefaultNameResolver(materialSvc *service.MaterialService) NameResolver {
	return func(ctx context.Context, name, tenantID string) ([]ResolveResult, error) {
		summaries, err := materialSvc.List(ctx, repository.MaterialFilter{
			TenantID: tenantID,
			Keyword:  name,
		})
		if err != nil {
			return nil, err
		}
		results := make([]ResolveResult, 0, len(summaries))
		for _, s := range summaries {
			score := computeMatchScore(name, s.Name, s.Spec)
			results = append(results, ResolveResult{
				MaterialID:   s.ID,
				Name:         s.Name,
				Spec:         s.Spec,
				DefaultUnit:  s.DefaultUnit,
				Score:        score,
				IsExactMatch: score >= 0.95,
			})
		}
		sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
		return results, nil
	}
}

// computeMatchScore returns a similarity score [0, 1] for name matching.
func computeMatchScore(input, dbName, dbSpec string) float64 {
	in := strings.TrimSpace(strings.ToLower(input))
	name := strings.TrimSpace(strings.ToLower(dbName))
	spec := strings.TrimSpace(strings.ToLower(dbSpec))

	full := name
	if spec != "" {
		full = name + " " + spec
	}

	if in == full || in == name {
		return 1.0
	}
	if strings.Contains(full, in) || strings.Contains(in, full) {
		return 0.8
	}
	return 0.0
}

// needsConfirmation returns true if there are multiple candidates with score >= 0.6.
func needsConfirmation(candidates []ResolveResult) bool {
	count := 0
	for _, c := range candidates {
		if c.Score >= 0.6 {
			count++
		}
	}
	return count > 1
}

// buildConfirmMessage formats candidate list for user disambiguation.
func buildConfirmMessage(name string, candidates []ResolveResult) string {
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
		b.WriteString(fmt.Sprintf(" — 单位: %s", c.DefaultUnit))
		b.WriteString("\n")
	}
	if len(candidates) > len(labels) {
		b.WriteString(fmt.Sprintf("  ...以及其他 %d 个\n", len(candidates)-len(labels)))
	}
	b.WriteString("请回复选项字母（如 A、B、C），或输入更精确的名称。")
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
