package kafka

import (
	"log/slog"

	slogcommon "github.com/samber/slog-common"
	"github.com/MikL9/observability/utils"
)

func DefaultConverter(loggerAttr []slog.Attr, record *slog.Record) map[string]any {
	// aggregate all attributes
	attrs := make([]slog.Attr, 0, record.NumAttrs()+len(loggerAttr))

	attrs = append(attrs, loggerAttr...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	payload := make(map[string]any, len(attrs)+3)

	for key, val := range slogcommon.AttrsToMap(attrs...) {
		if key == utils.ErrorKey {
			continue
		}
		payload[key] = val
	}
	payload["time"] = record.Time.UTC()
	payload["level"] = record.Level.String()
	payload["msg"] = record.Message

	return payload
}
