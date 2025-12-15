package logger

import (
	"log/slog"
	"reflect"
)

func Object(key string, v slog.LogValuer) slog.Attr {
	iv := reflect.ValueOf(v)
	if iv.Kind() == reflect.Ptr && iv.IsNil() {
		return slog.Any(key, nil)
	}
	if v == nil {
		return slog.Attr{}
	}
	return slog.Attr{Key: key, Value: v.LogValue()}
}

func Array[T slog.LogValuer](key string, v []T) slog.Attr {
	result := make([]map[string]any, 0, len(v))
	for _, attr := range v {
		if logVal := attr.LogValue(); logVal.Kind() == slog.KindGroup {
			result = append(result, resolveArray(logVal))
		}
	}
	return slog.Attr{Key: key, Value: slog.AnyValue(result)}
}

func resolveArray(val slog.Value) map[string]any {
	output := make(map[string]any)
	for _, attr := range val.Group() {
		if attr.Value.Kind() == slog.KindGroup {
			output[attr.Key] = resolveArray(attr.Value)
		} else {
			output[attr.Key] = attr.Value.Any()
		}
	}
	return output
}
