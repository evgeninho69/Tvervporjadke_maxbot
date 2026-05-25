package common

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/types"
	"context"
)

const (
	CTX_VALUES_KEY = types.ContextValueKey("ctx_values")
	UID_KEY = "uid"
	CORRELATION_ID_KEY = "correlation_id"
	ADMIN_DATA_KEY = "admin_data"
)

func CTXValuesExists(ctx context.Context) bool {
	switch ctx.Value(CTX_VALUES_KEY).(type) {
	case map[string]any:
		return true
	default:
		return false
	}
}

func GetValueFromContext(ctx context.Context, key string) any {
	values := ctx.Value(CTX_VALUES_KEY).(map[string]any)

	return values[key]
}

func SetValueIntoContext(ctx context.Context, key string, value any) {
	values := ctx.Value(CTX_VALUES_KEY).(map[string]any)

	values[key] = value
}

func GetUID(ctx context.Context) uint64 {
	uid := GetValueFromContext(ctx, UID_KEY)

	switch v := uid.(type) {
	case uint64:
		return v
	}

	return 0
}

func GetCorrelationID(ctx context.Context) string {
	correlationID := GetValueFromContext(ctx, CORRELATION_ID_KEY)
	
	switch v := correlationID.(type) {
	case string:
		return v
	}

	return ""
}

func GetAdminData(ctx context.Context) entity.Admin {
	adminData := GetValueFromContext(ctx, ADMIN_DATA_KEY)

	switch v := adminData.(type) {
	case entity.Admin:
		return v
	}

	return entity.Admin{}
}

