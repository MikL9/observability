package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/MikL9/observability/hide"
	"go.opentelemetry.io/otel/attribute"
)

type storageType string
type storageKeysKeyType string

const storageKeys storageKeysKeyType = "keys"

// SetContextAttr adding typed attributes into storage
func SetContextAttr(ctx context.Context, attrs ...slog.Attr) context.Context {
	keys, ok := ctx.Value(storageKeys).([]storageType)
	if !ok {
		keys = make([]storageType, 0, len(attrs))
	}
	for _, attr := range attrs {
		if !slices.Contains(keys, storageType(attr.Key)) {
			keys = append(keys, storageType(attr.Key))
			ctx = context.WithValue(ctx, storageKeysKeyType("keys"), keys)
		}
		ctx = context.WithValue(ctx, storageType(attr.Key), attr.Value.Any())
	}
	return ctx
}

func SetSlogGroupToString(attrs []slog.Attr, prefix string) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0, len(attrs))
	for _, at := range attrs {
		name := fmt.Sprintf("%s.%s", prefix, at.Key)
		switch at.Value.Kind() {
		case slog.KindBool:
			result = append(result, attribute.Bool(name, at.Value.Bool()))
		case slog.KindFloat64:
			result = append(result, attribute.Float64(name, at.Value.Float64()))
		case slog.KindInt64:
			result = append(result, attribute.Int64(name, at.Value.Int64()))
		case slog.KindGroup:
			result = append(result, SetSlogGroupToString(at.Value.Group(), name)...)
		default:
			result = append(result, attribute.String(name, at.Value.String()))
		}
	}
	return result
}

func ExportOtelAttributes(ctx context.Context) []attribute.KeyValue {
	keys, ok := ctx.Value(storageKeys).([]storageType)
	if !ok {
		return []attribute.KeyValue{}
	}
	attrs := make([]attribute.KeyValue, 0, len(keys))
	for _, k := range keys {
		key := string(k)
		switch val := ctx.Value(k).(type) {
		case string:
			attrs = append(attrs, attribute.String(key, hide.Hide(key, val)))
		case int:
			attrs = append(attrs, attribute.Int(key, val))
		case int64:
			attrs = append(attrs, attribute.Int64(key, val))
		case bool:
			attrs = append(attrs, attribute.Bool(key, val))
		case float64:
			attrs = append(attrs, attribute.Float64(key, val))
		case []slog.Attr:
			attrs = append(attrs, SetSlogGroupToString(val, key)...)
		default:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("{%+v}", val)))
		}
	}
	return attrs
}

func ExportSlogAttributes(ctx context.Context) []slog.Attr {
	keys, ok := ctx.Value(storageKeys).([]storageType)
	if !ok {
		return []slog.Attr{}
	}
	attrs := make([]slog.Attr, 0, len(keys))

	for _, k := range keys {
		key := string(k)
		switch val := ctx.Value(k).(type) {
		case string:
			attrs = append(attrs, slog.String(key, hide.Hide(key, val)))
		default:
			attrs = append(attrs, slog.Any(key, val))
		}
	}
	return attrs
}
