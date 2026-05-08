package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestScheduledTaskHandler_ListUpdateTriggerAndRuns(t *testing.T) {
	server, taskCenter, cleanup := newScheduledTaskTestServer(t, taskcenter.TaskDefinition{
		Code:              "demo_task",
		Name:              "Demo Task",
		Description:       "handler test task",
		DefaultCronSpec:   "0 8 * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 60,
		Run: func(ctx context.Context, actor taskcenter.Actor) (taskcenter.TaskResult, error) {
			return taskcenter.TaskResult{Summary: "demo completed", Payload: map[string]int{"done": 1}}, nil
		},
	})
	defer cleanup()
	defer taskCenter.Stop()

	list := performScheduledTaskJSONRequest(t, server, http.MethodGet, "/api/v1/scheduled-tasks", nil, http.StatusOK)
	items := list["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].(map[string]any)["code"] != "demo_task" {
		t.Fatalf("code = %v", items[0].(map[string]any)["code"])
	}

	updated := performScheduledTaskJSONRequest(t, server, http.MethodPatch, "/api/v1/scheduled-tasks/demo_task", map[string]any{
		"cron_spec":           "30 23 * * *",
		"enabled":             false,
		"run_timeout_seconds": 180,
	}, http.StatusOK)
	task := updated["data"].(map[string]any)
	if task["cron_spec"] != "30 23 * * *" {
		t.Fatalf("cron_spec = %v", task["cron_spec"])
	}
	if task["enabled"] != false {
		t.Fatalf("enabled = %v", task["enabled"])
	}

	trigger := performScheduledTaskJSONRequest(t, server, http.MethodPost, "/api/v1/scheduled-tasks/demo_task/trigger", map[string]any{}, http.StatusAccepted)
	runID := trigger["data"].(map[string]any)["id"]
	if runID == "" {
		t.Fatalf("run id = %v", runID)
	}

	if err := waitForScheduledTaskRuns(server, "/api/v1/scheduled-task-runs?task_code=demo_task&page=1&page_size=10", func(items []any) bool {
		for _, raw := range items {
			if raw.(map[string]any)["status"] == "success" {
				return true
			}
		}
		return false
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerHandler_LegacyCompatibility(t *testing.T) {
	server, taskCenter, cleanup := newScheduledTaskTestServer(t, taskcenter.TaskDefinition{
		Code:              taskcenter.LegacySchedulerTaskCode,
		Name:              "Expiry Notification",
		Description:       "legacy scheduler endpoint task",
		DefaultCronSpec:   "0 8 * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 60,
		Run: func(ctx context.Context, actor taskcenter.Actor) (taskcenter.TaskResult, error) {
			return taskcenter.TaskResult{Summary: "legacy completed"}, nil
		},
	})
	defer cleanup()
	defer taskCenter.Stop()

	status := performScheduledTaskJSONRequest(t, server, http.MethodGet, "/api/v1/scheduler/status", nil, http.StatusOK)
	if status["data"].(map[string]any)["state"] == "" {
		t.Fatalf("legacy state empty: %v", status)
	}

	trigger := performScheduledTaskJSONRequest(t, server, http.MethodPost, "/api/v1/scheduler/trigger", map[string]any{}, http.StatusAccepted)
	if trigger["data"].(map[string]any)["task_code"] != taskcenter.LegacySchedulerTaskCode {
		t.Fatalf("legacy task_code = %v", trigger["data"].(map[string]any)["task_code"])
	}
}

func newScheduledTaskTestServer(t *testing.T, def taskcenter.TaskDefinition) (*httpserver.Server, *taskcenter.TaskCenterService, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "scheduled-task-handler.db"),
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}

	uow := gormrepo.NewUnitOfWork(db)
	taskCenter := taskcenter.NewTaskCenterService(uow, def)
	if err := taskCenter.Start(context.Background()); err != nil {
		t.Fatalf("taskCenter.Start() error = %v", err)
	}

	scheduledTaskHandler := NewScheduledTaskHandler(taskCenter)
	schedulerHandler := NewSchedulerHandler(taskCenter)
	server := httpserver.New(appconfig.ServerConfig{}, scheduledTaskHandler.RegisterRoutes, schedulerHandler.RegisterRoutes)

	return server, taskCenter, func() { _ = sqlDB.Close() }
}

func performScheduledTaskJSONRequest(
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

func waitForScheduledTaskRuns(server *httpserver.Server, path string, match func(items []any) bool) error {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-User-Name", "Alice")
		req.Header.Set("X-User-ID", "user-1")
		req.Header.Set("X-Channel", "web")
		rec := httptest.NewRecorder()
		server.Engine().ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			var decoded map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err == nil {
				items := decoded["data"].(map[string]any)["items"].([]any)
				if match(items) {
					return nil
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
