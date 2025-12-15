package sentry

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/MikL9/observability/hide"
	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/utils"
)

var (
	name             = "observability/sentry-handler"
	sentryContextKey = "extra"
	LogLevels        = map[slog.Level]sentry.Level{
		slog.LevelDebug: sentry.LevelDebug,
		slog.LevelInfo:  sentry.LevelInfo,
		slog.LevelWarn:  sentry.LevelWarning,
		slog.LevelError: sentry.LevelError,
	}
)

func makeException(err *errors.Error) []sentry.Exception {
	exception := sentry.Exception{}

	var handled bool
	if !strings.Contains(err.Error(), "panic") {
		handled = true
	}
	exception.Mechanism = &sentry.Mechanism{Type: "go", Handled: &handled}

	exception.Type = err.Error()
	exception.Value = err.TypeName() + " " + err.Error()
	exception.Stacktrace = err.SentryStackTrace()
	if len(exception.Stacktrace.Frames) > 0 {
		exception.Module = exception.Stacktrace.Frames[0].Package
	}
	return []sentry.Exception{exception}
}

func makeEvent(ctx context.Context, record *slog.Record, env, release string) *sentry.Event {
	event := sentry.NewEvent()
	event.Environment = env
	event.Timestamp = record.Time.UTC()
	event.Level = LogLevels[record.Level]
	event.Logger = name
	event.Release = release

	if req, ok := ctx.Value("request").(*http.Request); ok {
		event.Request = sentry.NewRequest(req)
		event.Request.QueryString = hide.Hide("url", event.Request.QueryString)
	}

	record.Attrs(func(attr slog.Attr) bool {
		k := attr.Key
		v := attr.Value
		switch {
		case k == utils.ErrorKey && v.Kind() == slog.KindAny:
			if err, ok := v.Any().(*errors.Error); ok {
				event.Exception = makeException(err)
			}
		case attr.Key == string(utils.UserIDKey) && attr.Value.Kind() == slog.KindString:
			event.User.ID = attr.Value.String()
		case attr.Key == utils.TraceIDKey:
			event.Transaction = attr.Value.String()
		case k == utils.StacktraceKey:
			// exclude default stacktrace message
		default:
			if _, ok := event.Contexts[sentryContextKey]; !ok {
				event.Contexts[sentryContextKey] = make(map[string]any, 0)
			}
			event.Contexts[sentryContextKey][k] = v.String()
		}
		return true
	})
	return event
}
