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

func TestCategoryHandler_CRUDAndItemAssociation(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	createdCategory := performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "food",
	}, http.StatusCreated)
	categoryID, _ := createdCategory["id"].(string)
	if len(categoryID) != 10 {
		t.Fatalf("category ID length = %d", len(categoryID))
	}

	gotCategory := performJSONRequest(t, server, http.MethodGet, "/api/v1/categories/"+categoryID, "", nil, http.StatusOK)
	if gotCategory["name"] != "food" {
		t.Fatalf("name = %v", gotCategory["name"])
	}

	updatedCategory := performJSONRequest(t, server, http.MethodPut, "/api/v1/categories/"+categoryID, "", map[string]any{
		"name": "fresh-food",
	}, http.StatusOK)
	if updatedCategory["name"] != "fresh-food" {
		t.Fatalf("updated name = %v", updatedCategory["name"])
	}

	createdItem := performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name":        "eggs",
		"category_id": categoryID,
		"quantity":    12,
		"unit":        "box",
		"location":    "fridge",
	}, http.StatusCreated)
	itemID, _ := createdItem["id"].(string)
	if createdItem["category_id"] != categoryID {
		t.Fatalf("category_id = %v", createdItem["category_id"])
	}

	nestedCategory, ok := createdItem["category"].(map[string]any)
	if !ok {
		t.Fatalf("category = %#v", createdItem["category"])
	}
	if nestedCategory["name"] != "fresh-food" {
		t.Fatalf("category name = %v", nestedCategory["name"])
	}

	gotItem := performJSONRequest(t, server, http.MethodGet, "/api/v1/items/"+itemID, "", nil, http.StatusOK)
	if gotItem["category_id"] != categoryID {
		t.Fatalf("category_id = %v", gotItem["category_id"])
	}

	listedItems := performJSONRequest(t, server, http.MethodGet, "/api/v1/items?category_id="+categoryID, "", nil, http.StatusOK)
	if total, ok := listedItems["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("total = %v", listedItems["total"])
	}

	body := performJSONRequest(t, server, http.MethodDelete, "/api/v1/categories/"+categoryID, "", nil, http.StatusConflict)
	if body["error"] != repository.ErrCategoryInUse.Error() {
		t.Fatalf("error = %v", body["error"])
	}
}

func TestItemHandler_FullCRUDWithCategory(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	category := performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := category["id"].(string)

	created := performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name":        "milk",
		"category_id": categoryID,
		"quantity":    2,
		"unit":        "bottle",
		"location":    "shelf",
		"notes":       "test",
	}, http.StatusCreated)
	itemID, _ := created["id"].(string)
	if itemID == "" {
		t.Fatal("expected item ID")
	}

	expireAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	updated := performJSONRequest(t, server, http.MethodPut, "/api/v1/items/"+itemID, "", map[string]any{
		"name":        "milk-2",
		"category_id": categoryID,
		"quantity":    3,
		"expire_at":   expireAt,
	}, http.StatusOK)
	if updated["name"] != "milk-2" {
		t.Fatalf("updated name = %v", updated["name"])
	}

	performNoContentRequest(t, server, http.MethodDelete, "/api/v1/items/"+itemID, "", http.StatusNoContent)
	performJSONRequest(t, server, http.MethodGet, "/api/v1/items/"+itemID, "", nil, http.StatusNotFound)
	performNoContentRequest(t, server, http.MethodDelete, "/api/v1/categories/"+categoryID, "", http.StatusNoContent)
}

func TestItemHandler_RejectsInvalidCategoryID(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	body := performJSONRequest(t, server, http.MethodPost, "/api/v1/items", "", map[string]any{
		"name":        "rice",
		"category_id": "catbadid1",
	}, http.StatusBadRequest)
	if body["error"] != repository.ErrInvalidCategoryID.Error() {
		t.Fatalf("error = %v", body["error"])
	}
}

func TestCategoryHandler_TenantIsolationAndDuplicateNames(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "tenant-a", map[string]any{
		"name": "food",
	}, http.StatusCreated)
	performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "tenant-b", map[string]any{
		"name": "food",
	}, http.StatusCreated)

	body := performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "tenant-a", map[string]any{
		"name": "food",
	}, http.StatusConflict)
	if body["error"] != repository.ErrCategoryNameExists.Error() {
		t.Fatalf("error = %v", body["error"])
	}

	listed := performJSONRequest(t, server, http.MethodGet, "/api/v1/categories", "tenant-a", nil, http.StatusOK)
	if total, ok := listed["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("total = %v", listed["total"])
	}
}

func newTestServer(t *testing.T) (*httpserver.Server, func()) {
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

	categoryHandler := NewCategoryHandler(repository.NewCategoryRepository(db))
	itemHandler := NewItemHandler(repository.NewItemRepository(db))
	server := httpserver.New(appconfig.ServerConfig{}, categoryHandler.RegisterRoutes, itemHandler.RegisterRoutes)

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
