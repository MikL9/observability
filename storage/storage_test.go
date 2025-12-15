package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/MikL9/observability/hide"
	"go.opentelemetry.io/otel/attribute"
)

func keyString(v string) slog.Attr   { return slog.String("string", v) }
func keyInt(v int) slog.Attr         { return slog.Int("int", v) }
func keyFloat(v float64) slog.Attr   { return slog.Float64("float", v) }
func keyNew(v string) slog.Attr      { return slog.String("new", v) }
func keyPassword(v string) slog.Attr { return slog.String("password", v) }
func keyError(v any) slog.Attr       { return slog.Any("error", v) }
func setDefault() {
	hide.SetDefaultConverter(hide.NewConverter(
		hide.WithFullExcludeRule([]string{"token", "password"}),
	))
}

func TestSetContextAttr(t *testing.T) {
	ctx := context.Background()
	ctx = SetContextAttr(ctx,
		keyString("string"),
		keyInt(1),
		keyFloat(3.14),
	)
	ctx = SetContextAttr(ctx, keyNew("key"))

	expectedVal := map[string]any{
		"string": "string",
		"int":    int64(1),
		"float":  3.14,
		"new":    "key",
	}
	keys, ok := ctx.Value(storageKeys).([]storageType)
	require.True(t, ok, "keys must be not nil")
	for _, key := range keys {
		assert.Equal(t, expectedVal[string(key)], ctx.Value(key))
	}
}

func TestExportSlogAttributes(t *testing.T) {
	setDefault()
	err := errors.New("new error")
	ctx := SetContextAttr(context.Background(),
		keyString("value"),
		keyPassword("awesome"),
		keyInt(1),
		keyFloat(3.14),
		keyError(err),
	)

	attrs := ExportSlogAttributes(ctx)
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key > attrs[j].Key
	})

	expectedAttrs := []slog.Attr{
		slog.String("string", "value"),
		slog.String("password", "*******"),
		slog.Int("int", 1),
		slog.Float64("float", 3.14),
		slog.Any("error", err),
	}
	for i, attr := range attrs {
		assert.True(t, attr.Equal(expectedAttrs[i]), attr)
	}
}

func TestSetSlogGroupToString(t *testing.T) {
	group := slog.Group("group",
		slog.Int("age", 16),
		slog.String("name", "John"),
		slog.Bool("is_active", true),
		slog.Float64("pi", 3.14),
	)
	result := SetSlogGroupToString(group.Value.Group(), group.Key)
	assert.Equal(t, result, []attribute.KeyValue{
		attribute.Int64("group.age", 16),
		attribute.String("group.name", "John"),
		attribute.Bool("group.is_active", true),
		attribute.Float64("group.pi", 3.14),
	})

	// group in group
	group = slog.Group("group",
		slog.Group("0", slog.Int("age", 16)),
		slog.Group("1", slog.Int("age", 25)))
	result = SetSlogGroupToString(group.Value.Group(), group.Key)
	assert.Equal(t, result, []attribute.KeyValue{
		attribute.Int64("group.0.age", 16),
		attribute.Int64("group.1.age", 25),
	})
}

func TestExportOtelAttributes(t *testing.T) {
	setDefault()
	err := errors.New("new error")
	ctx := SetContextAttr(context.Background(),
		slog.String("string", "value"),
		slog.String("password", "*******"),
		slog.Int("int", 1),
		slog.Float64("float", 3.14),
		slog.Any("error", err),
		slog.Group("group", slog.Int("age", 16)),
	)
	attrs := ExportOtelAttributes(ctx)
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key > attrs[j].Key
	})

	expectedAttrs := []attribute.KeyValue{
		attribute.String("string", "value"),
		attribute.String("password", "*******"),
		attribute.Int("int", 1),
		attribute.Int64("group.age", 16),
		attribute.Float64("float", 3.14),
		attribute.String("error", fmt.Sprintf("{%+v}", err)),
	}
	for i, attr := range attrs {
		assert.True(t, attr.Valid())
		assert.Equal(t, expectedAttrs[i].Key, attr.Key)
		assert.Equal(t, expectedAttrs[i].Value, attr.Value)
	}
}
