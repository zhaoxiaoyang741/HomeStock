package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/outbound"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// OutboundHandler provides CRUD and test endpoints for outbound endpoints.
type OutboundHandler struct {
	configPath  string
	outboundMgr *outbound.Manager
}

// NewOutboundHandler creates an OutboundHandler.
func NewOutboundHandler(configPath string, outboundMgr *outbound.Manager) *OutboundHandler {
	return &OutboundHandler{configPath: configPath, outboundMgr: outboundMgr}
}

// RegisterRoutes mounts outbound endpoints.
func (h *OutboundHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/outbound/endpoints", h.ListEndpoints)
	api.POST("/outbound/endpoints", h.CreateEndpoint)
	api.PUT("/outbound/endpoints/:name", h.UpdateEndpoint)
	api.DELETE("/outbound/endpoints/:name", h.DeleteEndpoint)
	api.POST("/outbound/test", h.TestEndpoint)
}

func (h *OutboundHandler) ListEndpoints(c *gin.Context) {
	cfg := config.Get()
	httpresp.OK(c, cfg.Outbound.Endpoints)
}

type createEndpointRequest struct {
	Name    string `json:"name" binding:"required"`
	URL     string `json:"url" binding:"required"`
	Enabled *bool  `json:"enabled"`
}

func (h *OutboundHandler) CreateEndpoint(c *gin.Context) {
	var req createEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		for i, ep := range cfg.Outbound.Endpoints {
			if ep.Name == req.Name {
				cfg.Outbound.Endpoints[i] = config.EndpointConfig{
					Name: req.Name, URL: req.URL, Enabled: enabled,
				}
				return
			}
		}
		cfg.Outbound.Endpoints = append(cfg.Outbound.Endpoints, config.EndpointConfig{
			Name: req.Name, URL: req.URL, Enabled: enabled,
		})
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.Created(c, gin.H{"message": "endpoint saved"})
}

type updateEndpointRequest struct {
	URL     *string `json:"url"`
	Enabled *bool   `json:"enabled"`
}

func (h *OutboundHandler) UpdateEndpoint(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.Error(c, http.StatusBadRequest, "name is required")
		return
	}

	var req updateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		for i, ep := range cfg.Outbound.Endpoints {
			if ep.Name == name {
				if req.URL != nil {
					cfg.Outbound.Endpoints[i].URL = *req.URL
				}
				if req.Enabled != nil {
					cfg.Outbound.Endpoints[i].Enabled = *req.Enabled
				}
				return
			}
		}
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "endpoint updated"})
}

func (h *OutboundHandler) DeleteEndpoint(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.Error(c, http.StatusBadRequest, "name is required")
		return
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		endpoints := cfg.Outbound.Endpoints
		for i, ep := range endpoints {
			if ep.Name == name {
				cfg.Outbound.Endpoints = append(endpoints[:i], endpoints[i+1:]...)
				return
			}
		}
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "endpoint deleted"})
}

type testEndpointRequest struct {
	URL    string `json:"url" binding:"required"`
	Type   string `json:"type"`
	Body   string `json:"body"`
}

func (h *OutboundHandler) TestEndpoint(c *gin.Context) {
	var req testEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	event := outbound.OutboundEvent{
		Type:      outbound.EventType(req.Type),
		Timestamp: time.Now(),
		Payload:   req.Body,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	ep := outbound.NewHTTPEndpoint("test", req.URL)
	if err := ep.Send(ctx, event); err != nil {
		httpresp.Error(c, http.StatusBadGateway, "send test event failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "test event sent successfully"})
}
