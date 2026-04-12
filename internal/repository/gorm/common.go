package gormrepo

import "strings"

func normalizeTenantID(tenantID string) string {
	trimmed := strings.TrimSpace(tenantID)
	if trimmed == "" { return "default" }
	return trimmed
}
