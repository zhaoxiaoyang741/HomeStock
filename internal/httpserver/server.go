package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/webui"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// spaFS wraps an fs.FS for SPA single-page routing.
// If a file does not exist, it serves index.html instead of returning 404.
type spaFS struct {
	fs fs.FS
}

func (s *spaFS) Open(name string) (fs.File, error) {
	f, err := s.fs.Open(name)
	if err != nil {
		return s.fs.Open("index.html")
	}
	return f, nil
}

// AuthMiddleware is a Gin handler that validates JWT tokens and sets the Actor
// in context. A nil value means no auth is enforced (development fallback).
type AuthMiddleware gin.HandlerFunc

// RegisterRoutesFunc mounts application routes under /api/v1.
type RegisterRoutesFunc func(api *gin.RouterGroup)

// Server wraps the Gin engine and HTTP server lifecycle.
type Server struct {
	addr       string
	engine     *gin.Engine
	httpServer *http.Server
}

// New constructs a Gin-backed HTTP server with base middleware and routes.
// Public routes are mounted under /api/v1 without authentication.
// Protected routes are mounted on a sub-group that requires a valid JWT.
// Pass a nil authMiddleware to skip JWT enforcement (development fallback).
func New(cfg appconfig.ServerConfig, public []RegisterRoutesFunc, protected []RegisterRoutesFunc, authMiddleware AuthMiddleware) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery(), corsMiddleware(), requestLogger(), request.Middleware())

	api := engine.Group("/api/v1")
	api.GET("/health", healthHandler)

	for _, register := range public {
		if register != nil {
			register(api)
		}
	}

	if len(protected) > 0 {
		protectedGroup := api.Group("")
		if authMiddleware != nil {
			protectedGroup.Use(gin.HandlerFunc(authMiddleware))
		}
		for _, register := range protected {
			if register != nil {
				register(protectedGroup)
			}
		}
	}

	// SPA static file serving for embedded frontend.
	// NoRoute only fires when no API route matches.
	engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		staticFS := webui.DistFS()
		http.FileServer(http.FS(&spaFS{fs: staticFS})).ServeHTTP(c.Writer, c.Request)
	})

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
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the underlying HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func healthHandler(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		fields := map[string]any{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}
		// logger.InfoCF("http", "request completed", fields)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-Name, X-User-ID, X-Channel")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func normalizeAddr(port string) string {
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("0.0.0.0:%s", port)
}
