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

	srv := New(appconfig.ServerConfig{Port: "9090"}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	srv.Engine().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "{\"status\":\"ok\"}" {
		t.Fatalf("body = %q", body)
	}

	if srv.Addr() != "0.0.0.0:9090" {
		t.Fatalf("Addr() = %q", srv.Addr())
	}
}

func TestNew_mountsCustomRoutesUnderAPIV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	route := func(api *gin.RouterGroup) {
		api.GET("/materials", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"materials": []string{"rice"}})
		})
	}
	srv := New(appconfig.ServerConfig{}, []RegisterRoutesFunc{route}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/materials", nil)
	rec := httptest.NewRecorder()

	srv.Engine().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "{\"materials\":[\"rice\"]}" {
		t.Fatalf("body = %q", body)
	}
}

func TestNormalizeAddr_defaultsTo8080(t *testing.T) {
	if got := normalizeAddr(""); got != "0.0.0.0:8080" {
		t.Fatalf("normalizeAddr(\"\") = %q", got)
	}
}
