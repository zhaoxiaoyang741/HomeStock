package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// RegisterRoutesFunc mounts application routes under /api/v1.
type RegisterRoutesFunc func(api *gin.RouterGroup)

// Server wraps the Gin engine and HTTP server lifecycle.
type Server struct {
	addr       string
	engine     *gin.Engine
	httpServer *http.Server
}

// New constructs a Gin-backed HTTP server with base middleware and routes.
func New(cfg appconfig.ServerConfig, registrars ...RegisterRoutesFunc) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery(), corsMiddleware(), requestLogger(), request.Middleware())

	api := engine.Group("/api/v1")
	api.GET("/health", healthHandler)

	for _, register := range registrars {
		if register != nil {
			register(api)
		}
	}

	addr := normalizeAddr(cfg.Port)

	return &Server{
		addr:   addr,
		engine: engine,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Addr returns the listening address used by the HTTP server.
func (s *Server) Addr() string { return s.addr }

// Engine exposes the underlying Gin engine for additional setup or testing.
func (s *Server) Engine() *gin.Engine { return s.engine }

// Start blocks until the underlying HTTP server stops.
func (s *Server) Start() error {
	logger.InfoCF("http", "starting server", map[string]any{"addr": s.addr})
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) { return nil }
	return err
}

// Shutdown gracefully stops the underlying HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil { return nil }
	return s.httpServer.Shutdown(ctx)
}

func healthHandler(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath(); if path == "" { path = c.Request.URL.Path }
		fields := map[string]any{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
		}
		if len(c.Errors) > 0 { fields["errors"] = c.Errors.String() }
		logger.InfoCF("http", "request completed", fields)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-Name, X-User-ID, X-Channel")
		if c.Request.Method == http.MethodOptions { c.AbortWithStatus(http.StatusNoContent); return }
		c.Next()
	}
}

func normalizeAddr(port string) string {
	if port == "" { port = "8080" }
	return fmt.Sprintf("0.0.0.0:%s", port)
}
