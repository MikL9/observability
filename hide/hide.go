package hide

import (
	"log/slog"
)

func Hide(key, val string) string {
	if convert := defaultConverter.Load(); convert != nil {
		return convert.Hide(key, val)
	}
	return val
}

func Attrs(attrs []slog.Attr) []slog.Attr {
	for i, attr := range attrs {
		if attr.Value.Kind() == slog.KindString {
			val := attr.Value.String()
			attrs[i].Value = slog.StringValue(Hide(attr.Key, val))
		}
	}
	return attrs
}
