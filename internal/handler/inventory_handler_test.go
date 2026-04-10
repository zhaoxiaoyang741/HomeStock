package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestCategoryHandler_CRUDAndMaterialAssociation(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	createdCategory := performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "food",
	}, http.StatusCreated)
	categoryID, _ := createdCategory["id"].(string)

	createdMaterial := performJSONRequest(t, server, http.MethodPost, "/api/v1/materials", "", map[string]any{
		"name":         "eggs",
		"category_id":  categoryID,
		"default_unit": "box",
	}, http.StatusCreated)
	if createdMaterial["category_id"] != categoryID {
		t.Fatalf("category_id = %v", createdMaterial["category_id"])
	}

	body := performJSONRequest(t, server, http.MethodDelete, "/api/v1/categories/"+categoryID, "", nil, http.StatusConflict)
	if body["error"] != repository.ErrCategoryInUse.Error() {
		t.Fatalf("error = %v", body["error"])
	}
}

func TestInventoryFlow_InboundConsumeAdjustAndHistory(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	category := performJSONRequest(t, server, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := category["id"].(string)

	firstLot := performJSONRequest(t, server, http.MethodPost, "/api/v1/stock-lots/inbound", "", map[string]any{
		"name":         "milk",
		"spec":         "1L",
		"category_id":  categoryID,
		"quantity":     2,
		"unit":         "bottle",
		"location":     "fridge",
		"purchased_at": time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"expire_at":    time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}, http.StatusCreated)
	secondLot := performJSONRequest(t, server, http.MethodPost, "/api/v1/stock-lots/inbound", "", map[string]any{
		"name":         "milk",
		"spec":         "1L",
		"category_id":  categoryID,
		"quantity":     3,
		"unit":         "bottle",
		"location":     "fridge",
		"purchased_at": time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"expire_at":    time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}, http.StatusCreated)

	materials := performJSONRequest(t, server, http.MethodGet, "/api/v1/materials", "", nil, http.StatusOK)
	if total, ok := materials["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("materials total = %v", materials["total"])
	}
	materialList, _ := materials["materials"].([]any)
	material := materialList[0].(map[string]any)
	materialID, _ := material["id"].(string)
	if got, _ := material["lot_count"].(float64); int(got) != 2 {
		t.Fatalf("lot_count = %v", material["lot_count"])
	}

	consumeResult := performJSONRequest(t, server, http.MethodPost, "/api/v1/materials/"+materialID+"/consume", "", map[string]any{
		"quantity": 4,
		"reason":   "cook",
	}, http.StatusOK)
	consumedLots, _ := consumeResult["consumed_lots"].([]any)
	if len(consumedLots) != 2 {
		t.Fatalf("consumed_lots len = %d", len(consumedLots))
	}
	firstConsumed := consumedLots[0].(map[string]any)
	if firstConsumed["lot_id"] != firstLot["id"] {
		t.Fatalf("expected first lot to be consumed first, got %v", firstConsumed["lot_id"])
	}

	adjusted := performJSONRequest(t, server, http.MethodPost, "/api/v1/stock-lots/"+secondLot["id"].(string)+"/adjust", "", map[string]any{
		"target_quantity": 5,
		"reason":          "count",
		"remark":          "restock correction",
	}, http.StatusOK)
	if adjusted["quantity_on_hand"] != 5.0 {
		t.Fatalf("quantity_on_hand = %v", adjusted["quantity_on_hand"])
	}

	movements := performJSONRequest(t, server, http.MethodGet, "/api/v1/stock-movements", "", nil, http.StatusOK)
	if total, ok := movements["total"].(float64); !ok || int(total) != 5 {
		t.Fatalf("movements total = %v", movements["total"])
	}
}

func TestStockLotUpdateDoesNotTouchOtherLots(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	firstLot := performJSONRequest(t, server, http.MethodPost, "/api/v1/stock-lots/inbound", "", map[string]any{
		"name":     "rice",
		"spec":     "5kg",
		"quantity": 1,
		"unit":     "bag",
		"location": "cabinet",
	}, http.StatusCreated)
	secondLot := performJSONRequest(t, server, http.MethodPost, "/api/v1/stock-lots/inbound", "", map[string]any{
		"name":     "rice",
		"spec":     "5kg",
		"quantity": 1,
		"unit":     "bag",
		"location": "storage",
	}, http.StatusCreated)

	updated := performJSONRequest(t, server, http.MethodPut, "/api/v1/stock-lots/"+firstLot["id"].(string), "", map[string]any{
		"location": "kitchen",
		"notes":    "opened",
	}, http.StatusOK)
	if updated["location"] != "kitchen" {
		t.Fatalf("updated location = %v", updated["location"])
	}

	listed := performJSONRequest(t, server, http.MethodGet, "/api/v1/stock-lots?keyword=rice", "", nil, http.StatusOK)
	lots, _ := listed["lots"].([]any)
	if len(lots) != 2 {
		t.Fatalf("lots len = %d", len(lots))
	}
	for _, raw := range lots {
		lot := raw.(map[string]any)
		if lot["id"] == secondLot["id"] && lot["location"] != "storage" {
			t.Fatalf("second lot location changed unexpectedly: %v", lot["location"])
		}
	}
}

func TestMaterialHandler_RejectsInvalidCategoryID(t *testing.T) {
	server, cleanup := newTestServer(t)
	defer cleanup()

	body := performJSONRequest(t, server, http.MethodPost, "/api/v1/materials", "", map[string]any{
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
}

func newTestServer(t *testing.T) (*httpserver.Server, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "inventory.db"),
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}

	auditRepo := repository.NewAuditLogRepository(db)
	categoryHandler := NewCategoryHandler(repository.NewCategoryRepository(db), auditRepo)
	materialHandler := NewMaterialHandler(
		db,
		repository.NewMaterialRepository(db),
		repository.NewStockLotRepository(db),
		repository.NewStockMovementRepository(db),
		auditRepo,
	)
	stockLotHandler := NewStockLotHandler(
		db,
		repository.NewStockLotRepository(db),
		repository.NewMaterialRepository(db),
		repository.NewStockMovementRepository(db),
		auditRepo,
	)
	stockMovementHandler := NewStockMovementHandler(repository.NewStockMovementRepository(db))
	server := httpserver.New(
		appconfig.ServerConfig{},
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
	)

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
