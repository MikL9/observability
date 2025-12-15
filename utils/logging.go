package utils

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/MikL9/observability/hide"
	"go.opentelemetry.io/otel/trace"
)

func SetTraceSpanIDToAttrs(ctx context.Context, attrs []slog.Attr) []slog.Attr {
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.SpanID().IsValid() && spanCtx.TraceID().IsValid() {
		attrs = append(attrs,
			slog.String(SpanIDKey, spanCtx.SpanID().String()),
			slog.String(TraceIDKey, spanCtx.TraceID().String()),
		)
	}
	return attrs
}

type ContextKey string

const (
	UserIDKey     ContextKey = "user_id"
	ErrorKey                 = "error_msg"
	StacktraceKey            = "stacktrace"
	SpanIDKey                = "span_id"
	TraceIDKey               = "trace_id"
)

// KeyRequest должен использоваться до исполнения запроса,
// так как body будет уже прочтен и будет пустым
func KeyRequest(r *http.Request, stringBody bool) slog.Attr {
	attrs := []slog.Attr{
		slog.String("remote_addr", GetRealIP(r)),
		slog.String("url", r.RequestURI),
		slog.String("path", r.URL.Path),
		slog.String("proto", r.Proto),
		slog.String("host", r.Host),
		slog.Int64("body_length", r.ContentLength),
		slog.String("method", r.Method),
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		attrs = append(attrs, hide.JSON("body", GetRequestBodyCopy(r), 10*Kilobyte, stringBody))
	}
	return slog.Attr{Key: "request", Value: slog.GroupValue(attrs...)}
}
func KeyResponse(resp *ResponseData, stringBody bool) slog.Attr {
	return slog.Group("response",
		slog.Int("status", resp.Status),
		slog.Int("content_length", len(resp.Body)),
		hide.JSON("body", resp.Body, 10*Kilobyte, stringBody),
	)
}
func KeyPanic(cause string, skip int) slog.Attr {
	pc, file, line, _ := runtime.Caller(skip + 1)
	return slog.Group("panic",
		slog.String("recover-error", cause),
		slog.String("file", file),
		slog.Int("line", line),
		slog.String("function", runtime.FuncForPC(pc).Name()),
		KeyStacktrace(string(debug.Stack())),
	)
}
func KeyDuration(v time.Time) slog.Attr { return slog.String("duration", time.Since(v).String()) }

func KeyAttempt(v int) slog.Attr    { return slog.Int("attempt", v) }
func KeyMaxAttempt(v int) slog.Attr { return slog.Int("max_attempt", v) }

func KeyStacktrace(v string) slog.Attr { return slog.String(StacktraceKey, v) }
func KeyError(v error) slog.Attr       { return slog.Attr{Key: ErrorKey, Value: slog.AnyValue(v)} }

func AppType(v string) slog.Attr {
	return slog.String("app_type", v)
}
func AppVersion(v string) slog.Attr { return slog.String("app_version", v) }
