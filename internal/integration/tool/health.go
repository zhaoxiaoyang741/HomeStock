package tool

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/reply"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
)

// HealthTool checks whether the HomeStock backend service is reachable.
type HealthTool struct {
	BaseURL string
	Client  *http.Client
}

func (h *HealthTool) CheckService(ctx context.Context, actor service.Actor, _ map[string]any) (string, error) {
	if h.Client == nil {
		h.Client = &http.Client{Timeout: 5 * time.Second}
	}
	rc := reply.ForChannel(actor.Channel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.BaseURL+"/api/v1/health", nil)
	if err != nil {
		return reply.Error(rc, "健康检查失败：请求构造错误"), nil
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return reply.Error(rc, "服务不可达，请检查后端服务是否正常运行。"), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return reply.Success(rc, "HomeStock 服务运行正常。"), nil
	}
	return reply.Warning(rc, fmt.Sprintf("服务响应异常，状态码: %d", resp.StatusCode)), nil
}

func HealthToolDefinition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunctionDefinition{
			Name:        "check_homestock_service",
			Description: "检查HomeStock后端服务的健康状态。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func RegisterHealthTool(d *Dispatcher, baseURL string) {
	ht := &HealthTool{BaseURL: baseURL}
	d.Register("check_homestock_service", func(ctx context.Context, actor service.Actor, args map[string]any) (string, error) {
		return ht.CheckService(ctx, actor, args)
	})
}
