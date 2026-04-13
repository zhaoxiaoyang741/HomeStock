package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestSystemSettingsHandler_GetUsesConfigDefaults(t *testing.T) {
	server, cleanup := newSystemSettingsTestServer(t, &appconfig.Config{
		Server: appconfig.ServerConfig{},
		Database: appconfig.DatabaseConfig{
			Driver: "sqlite",
		},
		Notify: appconfig.NotifyConfig{},
		Scheduler: appconfig.SchedulerConfig{
			RemindDays: 5,
			CheckTime:  "09:30",
		},
	})
	defer cleanup()

	body := performSystemSettingsJSONRequest(t, server, http.MethodGet, "/api/v1/settings/system", nil, http.StatusOK)
	data := body["data"].(map[string]any)
	if data["version"] != 0.0 {
		t.Fatalf("version = %v", data["version"])
	}
	reminder := data["reminder"].(map[string]any)
	if reminder["remind_days"] != 5.0 {
		t.Fatalf("remind_days = %v", reminder["remind_days"])
	}
	if reminder["check_time"] != "09:30" {
		t.Fatalf("check_time = %v", reminder["check_time"])
	}
}

func TestSystemSettingsHandler_UpdatePersistsAndAudits(t *testing.T) {
	server, cleanup := newSystemSettingsTestServer(t, &appconfig.Config{
		Server: appconfig.ServerConfig{},
		Database: appconfig.DatabaseConfig{
			Driver: "sqlite",
		},
		Scheduler: appconfig.SchedulerConfig{
			RemindDays: 3,
			CheckTime:  "08:00",
		},
	})
	defer cleanup()

	updated := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 7,
			"check_time":  "10:15",
		},
		"notify": map[string]any{
			"enabled":             true,
			"feishu_webhook_mode": "replace",
			"feishu_webhook":      "https://open.feishu.cn/hook/test123456",
		},
	}, http.StatusOK)
	updatedData := updated["data"].(map[string]any)
	if updatedData["version"] != 1.0 {
		t.Fatalf("version = %v", updatedData["version"])
	}
	if updatedBy := updatedData["updated_by"].(map[string]any); updatedBy["user_name"] != "Alice" {
		t.Fatalf("updated_by.user_name = %v", updatedBy["user_name"])
	}

	reloaded := performSystemSettingsJSONRequest(t, server, http.MethodGet, "/api/v1/settings/system", nil, http.StatusOK)
	reloadedData := reloaded["data"].(map[string]any)
	reloadedReminder := reloadedData["reminder"].(map[string]any)
	if reloadedReminder["remind_days"] != 7.0 {
		t.Fatalf("reloaded remind_days = %v", reloadedReminder["remind_days"])
	}
	notify := reloadedData["notify"].(map[string]any)
	if notify["feishu_webhook_configured"] != true {
		t.Fatalf("feishu_webhook_configured = %v", notify["feishu_webhook_configured"])
	}
	if notify["feishu_webhook_masked"] == "" {
		t.Fatalf("feishu_webhook_masked = empty")
	}

	keep := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 1,
		"reminder": map[string]any{
			"remind_days": 9,
			"check_time":  "11:30",
		},
		"notify": map[string]any{
			"enabled":             true,
			"feishu_webhook_mode": "keep",
		},
	}, http.StatusOK)
	if keep["data"].(map[string]any)["version"] != 2.0 {
		t.Fatalf("keep version = %v", keep["data"].(map[string]any)["version"])
	}

	clear := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 2,
		"reminder": map[string]any{
			"remind_days": 9,
			"check_time":  "11:30",
		},
		"notify": map[string]any{
			"enabled":             false,
			"feishu_webhook_mode": "clear",
		},
	}, http.StatusOK)
	clearNotify := clear["data"].(map[string]any)["notify"].(map[string]any)
	if clearNotify["feishu_webhook_configured"] != false {
		t.Fatalf("clear feishu_webhook_configured = %v", clearNotify["feishu_webhook_configured"])
	}

	auditLogs := performSystemSettingsJSONRequest(t, server, http.MethodGet, "/api/v1/audit-logs?page=1&page_size=20", nil, http.StatusOK)
	items := auditLogs["data"].(map[string]any)["items"].([]any)
	found := false
	for _, raw := range items {
		log := raw.(map[string]any)
		if log["entity_type"] == "system_setting" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected system_setting audit log, got %v", items)
	}
}

func TestSystemSettingsHandler_ValidatesAndDetectsConflicts(t *testing.T) {
	server, cleanup := newSystemSettingsTestServer(t, &appconfig.Config{
		Server: appconfig.ServerConfig{},
		Database: appconfig.DatabaseConfig{
			Driver: "sqlite",
		},
		Scheduler: appconfig.SchedulerConfig{
			RemindDays: 3,
			CheckTime:  "08:00",
		},
	})
	defer cleanup()

	invalidTime := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 3,
			"check_time":  "25:61",
		},
		"notify": map[string]any{
			"enabled":             false,
			"feishu_webhook_mode": "keep",
		},
	}, http.StatusBadRequest)
	if invalidTime["message"] == "" {
		t.Fatalf("invalidTime message empty")
	}

	invalidDays := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 31,
			"check_time":  "08:00",
		},
		"notify": map[string]any{
			"enabled":             false,
			"feishu_webhook_mode": "keep",
		},
	}, http.StatusBadRequest)
	if invalidDays["message"] == "" {
		t.Fatalf("invalidDays message empty")
	}

	missingWebhook := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 3,
			"check_time":  "08:00",
		},
		"notify": map[string]any{
			"enabled":             true,
			"feishu_webhook_mode": "keep",
		},
	}, http.StatusBadRequest)
	if missingWebhook["message"] == "" {
		t.Fatalf("missingWebhook message empty")
	}

	_ = performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 4,
			"check_time":  "07:45",
		},
		"notify": map[string]any{
			"enabled":             true,
			"feishu_webhook_mode": "replace",
			"feishu_webhook":      "https://open.feishu.cn/hook/conflict1234",
		},
	}, http.StatusOK)

	conflict := performSystemSettingsJSONRequest(t, server, http.MethodPut, "/api/v1/settings/system", map[string]any{
		"version": 0,
		"reminder": map[string]any{
			"remind_days": 5,
			"check_time":  "09:00",
		},
		"notify": map[string]any{
			"enabled":             true,
			"feishu_webhook_mode": "keep",
		},
	}, http.StatusConflict)
	if conflict["message"] != "system settings version conflict" {
		t.Fatalf("conflict message = %v", conflict["message"])
	}
}

func newSystemSettingsTestServer(t *testing.T, cfg *appconfig.Config) (*httpserver.Server, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfgCopy := *cfg
	cfgCopy.Database.DSN = filepath.Join(t.TempDir(), "settings.db")
	if cfgCopy.Database.Driver == "" {
		cfgCopy.Database.Driver = "sqlite"
	}

	db, err := database.OpenAndMigrate(cfgCopy.Database)
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}

	uow := gormrepo.NewUnitOfWork(db)
	auditHandler := NewAuditLogHandler(service.NewAuditService(uow))
	settingsHandler := NewSystemSettingsHandler(service.NewSystemSettingsService(uow, &cfgCopy))
	server := httpserver.New(cfgCopy.Server, auditHandler.RegisterRoutes, settingsHandler.RegisterRoutes)

	return server, func() { _ = sqlDB.Close() }
}

func performSystemSettingsJSONRequest(
	t *testing.T,
	server *httpserver.Server,
	method string,
	path string,
	body any,
	wantStatus int,
) map[string]any {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-User-Name", "Alice")
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Channel", "web")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
