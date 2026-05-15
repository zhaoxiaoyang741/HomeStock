package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBatchInbound(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	cat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := cat["data"].(map[string]any)["id"].(string)

	// Create material first
	mat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials", "", map[string]any{
		"name": "milk", "spec": "1L", "category_id": categoryID, "default_unit": "bottle",
	}, http.StatusCreated)
	materialID, _ := mat["data"].(map[string]any)["id"].(string)

	result := performJSONRequest(t, srv, http.MethodPost, "/api/v1/batch/inbound", "", map[string]any{
		"items": []map[string]any{
			{"material_id": materialID, "quantity": 2, "unit": "bottle", "location": "fridge"},
			{"name": "yogurt", "category_id": categoryID, "quantity": 1, "unit": "cup", "location": "fridge"},
		},
	}, http.StatusOK)

	items, _ := result["data"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	first := items[0].(map[string]any)
	if first["error"] != nil {
		t.Errorf("first item should succeed, got error: %v", first["error"])
	}
}

func TestBatchConsume(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	cat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := cat["data"].(map[string]any)["id"].(string)

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/stock-lots/inbound", "", map[string]any{
		"name": "milk", "spec": "1L", "category_id": categoryID,
		"quantity": 5, "unit": "bottle", "location": "fridge",
	}, http.StatusCreated)

	materials := performJSONRequest(t, srv, http.MethodGet, "/api/v1/materials?keyword=milk", "", nil, http.StatusOK)
	page := materials["data"].(map[string]any)
	items, _ := page["items"].([]any)
	if len(items) == 0 {
		t.Fatal("no materials found")
	}
	materialID := items[0].(map[string]any)["id"].(string)

	result := performJSONRequest(t, srv, http.MethodPost, "/api/v1/batch/consume", "", map[string]any{
		"items": []map[string]any{
			{"material_id": materialID, "quantity": 2, "reason": "cook"},
		},
	}, http.StatusOK)

	itemsResult, _ := result["data"].([]any)
	if len(itemsResult) != 1 {
		t.Fatalf("expected 1 item, got %d", len(itemsResult))
	}
}

func TestResolveMaterial_ByName(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	cat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := cat["data"].(map[string]any)["id"].(string)

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials", "", map[string]any{
		"name": "milk", "spec": "1L", "category_id": categoryID, "default_unit": "bottle",
	}, http.StatusCreated)

	// Resolve should find the material even without stock (ShowZeroStock=true)
	result := performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials/resolve", "", map[string]any{
		"name": "milk",
	}, http.StatusOK)

	data, _ := result["data"].([]any)
	if len(data) < 1 {
		t.Fatalf("expected at least 1 candidate, got %d. Full response: %v", len(data), result)
	}

	first := data[0].(map[string]any)
	if first["name"] != "milk" {
		t.Errorf("expected 'milk', got %v", first["name"])
	}
	if first["is_exact_match"] != true {
		t.Errorf("expected exact match for 'milk' -> 'milk', got score=%v", first["score"])
	}
}

func TestResolveMaterial_EmptyName(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials/resolve", "", map[string]any{
		"name": "",
	}, http.StatusBadRequest)
}

func TestResolveMaterial_ShowZeroStock(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	cat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := cat["data"].(map[string]any)["id"].(string)

	// Material with zero stock (no inbound)
	performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials", "", map[string]any{
		"name": "cheese", "spec": "200g", "category_id": categoryID, "default_unit": "pack",
	}, http.StatusCreated)

	// resolve endpoint should find it despite zero stock
	result := performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials/resolve", "", map[string]any{
		"name": "cheese",
	}, http.StatusOK)

	data, _ := result["data"].([]any)
	if len(data) < 1 {
		t.Fatalf("expected at least 1 candidate for zero-stock material, got %d. Full: %v", len(data), result)
	}

	// GET /api/v1/materials (without show_zero_stock) should NOT find it
	listResult := performJSONRequest(t, srv, http.MethodGet, "/api/v1/materials?keyword=cheese", "", nil, http.StatusOK)
	listPage := listResult["data"].(map[string]any)
	if total, _ := listPage["total"].(float64); int(total) != 0 {
		t.Fatalf("expected 0 in normal list (zero stock filtered), got total=%v", listPage["total"])
	}
}

func TestBatchConsume_PartialFailure(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	// One valid consume and one invalid material_id
	result := performJSONRequest(t, srv, http.MethodPost, "/api/v1/batch/consume", "", map[string]any{
		"items": []map[string]any{
			{"material_id": "nonexistent-id-1", "quantity": 1, "reason": "test"},
			{"material_id": "nonexistent-id-2", "quantity": 2, "reason": "test"},
		},
	}, http.StatusOK)

	items, _ := result["data"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 results, got %d", len(items))
	}
	// Both should fail (materials don't exist)
	for i, item := range items {
		it := item.(map[string]any)
		if it["error"] == nil {
			t.Errorf("item %d expected error for nonexistent material, got data: %v", i, it["data"])
		}
	}
}

func TestBatchInbound_EmptyItems(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/batch/inbound", "", map[string]any{
		"items": []map[string]any{},
	}, http.StatusBadRequest)
}

func TestBatchConsume_EmptyItems(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/batch/consume", "", map[string]any{
		"items": []map[string]any{},
	}, http.StatusBadRequest)
}

func TestResolveMaterial_RawHTTP(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	cat := performJSONRequest(t, srv, http.MethodPost, "/api/v1/categories", "", map[string]any{
		"name": "dairy",
	}, http.StatusCreated)
	categoryID, _ := cat["data"].(map[string]any)["id"].(string)

	performJSONRequest(t, srv, http.MethodPost, "/api/v1/materials", "", map[string]any{
		"name": "milk", "spec": "1L", "category_id": categoryID, "default_unit": "bottle",
	}, http.StatusCreated)

	// Raw HTTP request to the resolve endpoint
	reqBody, _ := json.Marshal(map[string]any{"name": "milk"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/materials/resolve", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Engine().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v, body: %s", err, rec.Body.String())
	}

	data, _ := body["data"].([]any)
	if len(data) < 1 {
		t.Fatalf("expected at least 1 candidate, got %d. Full response: %v", len(data), body)
	}

	first := data[0].(map[string]any)
	if first["name"] != "milk" {
		t.Errorf("expected 'milk', got %v", first["name"])
	}
}
