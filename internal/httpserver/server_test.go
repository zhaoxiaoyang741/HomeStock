package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestNew_registersHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := New(appconfig.ServerConfig{Port: "9090"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	srv.Engine().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "{\"status\":\"ok\"}" {
		t.Fatalf("body = %q", body)
	}

	if srv.Addr() != ":9090" {
		t.Fatalf("Addr() = %q", srv.Addr())
	}
}

func TestNew_mountsCustomRoutesUnderAPIV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := New(appconfig.ServerConfig{}, func(api *gin.RouterGroup) {
		api.GET("/items", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"items": []string{"rice"}})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	rec := httptest.NewRecorder()

	srv.Engine().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "{\"items\":[\"rice\"]}" {
		t.Fatalf("body = %q", body)
	}
}

func TestNormalizeAddr_defaultsTo8080(t *testing.T) {
	if got := normalizeAddr(""); got != ":8080" {
		t.Fatalf("normalizeAddr(\"\") = %q", got)
	}
}
