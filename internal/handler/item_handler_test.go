package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestItemHandler_CreateAndList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	itemHandler := NewItemHandler(repository.NewItemRepository(db))
	server := httpserver.New(appconfig.ServerConfig{}, itemHandler.RegisterRoutes)

	createBody := map[string]any{
		"name":     "鸡蛋",
		"category": "食材",
		"quantity": 12,
		"location": "冰箱",
		"notes":    "一盒",
	}

	body, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/items", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	server.Engine().ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %q", createRec.Code, createRec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	if created["name"] != "鸡蛋" {
		t.Fatalf("created name = %v", created["name"])
	}
	if created["tenant_id"] != "default" {
		t.Fatalf("created tenant_id = %v", created["tenant_id"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	listRec := httptest.NewRecorder()

	server.Engine().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %q", listRec.Code, listRec.Body.String())
	}

	var listed struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}

	if listed.Total != 1 {
		t.Fatalf("total = %d", listed.Total)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("len(items) = %d", len(listed.Items))
	}
	if listed.Items[0]["location"] != "冰箱" {
		t.Fatalf("location = %v", listed.Items[0]["location"])
	}
}

func TestItemHandler_ListRespectsTenantHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	itemHandler := NewItemHandler(repository.NewItemRepository(db))
	server := httpserver.New(appconfig.ServerConfig{}, itemHandler.RegisterRoutes)

	for _, tc := range []struct {
		tenant string
		name   string
	}{
		{tenant: "tenant-a", name: "大米"},
		{tenant: "tenant-b", name: "牛奶"},
	} {
		body, err := json.Marshal(map[string]any{"name": tc.name})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tc.tenant)
		rec := httptest.NewRecorder()
		server.Engine().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d, body = %q", rec.Code, rec.Body.String())
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	listReq.Header.Set("X-Tenant-ID", "tenant-a")
	listRec := httptest.NewRecorder()
	server.Engine().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %q", listRec.Code, listRec.Body.String())
	}

	var listed struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}

	if listed.Total != 1 {
		t.Fatalf("total = %d", listed.Total)
	}
	if listed.Items[0]["name"] != "大米" {
		t.Fatalf("name = %v", listed.Items[0]["name"])
	}
}
