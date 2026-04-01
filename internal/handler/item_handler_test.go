package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestItemHandler_CreateGetUpdateDeleteFlow(t *testing.T) {
	server, cleanup := newItemTestServer(t)
	defer cleanup()

	createBody := map[string]any{
		"name":     "eggs",
		"category": "food",
		"quantity": 12,
		"unit":     "box",
		"location": "fridge",
		"notes":    "first batch",
	}

	created := performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", createBody, http.StatusCreated)
	itemID, _ := created["id"].(string)
	if itemID == "" {
		t.Fatal("expected created item ID")
	}

	got := performJSONRequest(t, server, http.MethodGet, "/api/v1/items/"+itemID, "", nil, http.StatusOK)
	if got["name"] != "eggs" {
		t.Fatalf("name = %v", got["name"])
	}

	expireAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	updated := performJSONRequest(t, server, http.MethodPut, "/api/v1/items/"+itemID, "", map[string]any{
		"name":      "milk",
		"quantity":  2,
		"location":  "shelf",
		"expire_at": expireAt,
	}, http.StatusOK)
	if updated["name"] != "milk" {
		t.Fatalf("updated name = %v", updated["name"])
	}
	if updated["location"] != "shelf" {
		t.Fatalf("updated location = %v", updated["location"])
	}

	listed := performJSONRequest(t, server, http.MethodGet, "/api/v1/items", "", nil, http.StatusOK)
	total, ok := listed["total"].(float64)
	if !ok || int(total) != 1 {
		t.Fatalf("total = %v", listed["total"])
	}

	items, ok := listed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", listed["items"])
	}

	performNoContentRequest(t, server, http.MethodDelete, "/api/v1/items/"+itemID, "", http.StatusNoContent)
	performJSONRequest(t, server, http.MethodGet, "/api/v1/items/"+itemID, "", nil, http.StatusNotFound)
}

func TestItemHandler_ListRespectsTenantHeader(t *testing.T) {
	server, cleanup := newItemTestServer(t)
	defer cleanup()

	performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "tenant-a", map[string]any{
		"name": "rice",
	}, http.StatusCreated)
	performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "tenant-b", map[string]any{
		"name": "milk",
	}, http.StatusCreated)

	listed := performJSONRequest(t, server, http.MethodGet, "/api/v1/items", "tenant-a", nil, http.StatusOK)
	total, ok := listed["total"].(float64)
	if !ok || int(total) != 1 {
		t.Fatalf("total = %v", listed["total"])
	}

	items, ok := listed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", listed["items"])
	}

	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("item = %#v", items[0])
	}
	if item["name"] != "rice" {
		t.Fatalf("name = %v", item["name"])
	}
	if item["tenant_id"] != "tenant-a" {
		t.Fatalf("tenant_id = %v", item["tenant_id"])
	}
}

func TestItemHandler_ListSupportsFilters(t *testing.T) {
	server, cleanup := newItemTestServer(t)
	defer cleanup()

	performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name":     "rice",
		"category": "dry",
		"location": "pantry",
	}, http.StatusCreated)
	performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name":     "milk",
		"category": "cold",
		"location": "fridge",
	}, http.StatusCreated)

	listed := performJSONRequest(t, server, http.MethodGet, "/api/v1/items?location=fridge&category=cold", "", nil, http.StatusOK)
	total, ok := listed["total"].(float64)
	if !ok || int(total) != 1 {
		t.Fatalf("total = %v", listed["total"])
	}
}

func TestItemHandler_UpdateRejectsInvalidExpireAt(t *testing.T) {
	server, cleanup := newItemTestServer(t)
	defer cleanup()

	created := performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name": "rice",
	}, http.StatusCreated)
	itemID, _ := created["id"].(string)

	body := performJSONRequest(t, server, http.MethodPut, "/api/v1/items/"+itemID, "", map[string]any{
		"expire_at": "not-a-date",
	}, http.StatusBadRequest)
	if body["error"] != "invalid expire_at, must be RFC3339" {
		t.Fatalf("error = %v", body["error"])
	}
}

func newItemTestServer(t *testing.T) (*httpserver.Server, func()) {
	t.Helper()
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

	itemHandler := NewItemHandler(repository.NewItemRepository(db))
	server := httpserver.New(appconfig.ServerConfig{}, itemHandler.RegisterRoutes)

	return server, func() {
		_ = sqlDB.Close()
	}
}

func performJSONRequest(
	t *testing.T,
	server *httpserver.Server,
	method string,
	path string,
	tenantID string,
	body any,
	wantStatus int,
) map[string]any {
	t.Helper()

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}

	rec := httptest.NewRecorder()
	server.Engine().ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, body = %q", method, path, rec.Code, rec.Body.String())
	}

	if rec.Body.Len() == 0 {
		return map[string]any{}
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %q", err, rec.Body.String())
	}

	return decoded
}

func performNoContentRequest(
	t *testing.T,
	server *httpserver.Server,
	method string,
	path string,
	tenantID string,
	wantStatus int,
) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}

	rec := httptest.NewRecorder()
	server.Engine().ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, body = %q", method, path, rec.Code, rec.Body.String())
	}
}
