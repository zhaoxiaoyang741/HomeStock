package tasks

import (
	"context"
	"fmt"

	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
)

// NewBackupTaskDefinition is a template for future manually-registered tasks.
func NewBackupTaskDefinition() taskcenter.TaskDefinition {
	return taskcenter.TaskDefinition{
		Code:              "backup_database",
		Name:              "Backup Database",
		Description:       "Example task definition for future backup automation.",
		DefaultCronSpec:   "0 2 * * *",
		DefaultEnabled:    false,
		RunTimeoutSeconds: 600,
		Run: func(ctx context.Context, actor taskcenter.Actor) (taskcenter.TaskResult, error) {
			return taskcenter.TaskResult{Summary: "backup task is not implemented"}, fmt.Errorf("backup task is not implemented")
		},
	}
}
