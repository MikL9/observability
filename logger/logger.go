package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/MikL9/observability/hide"
	"github.com/MikL9/observability/logger/errors"
	handlerStorage "github.com/MikL9/observability/logger/handlers/storage"
	"github.com/MikL9/observability/utils"
)

var timeNow = time.Now

func SetupLogger(h slog.Handler) error {
	slog.SetDefault(
		slog.New(
			handlerStorage.NewHandler(h),
		),
	)
	return nil
}

func Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logger := slog.Default()
	if !logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	r := slog.NewRecord(timeNow(), level, msg, pcs[0])

	attrs = utils.SetTraceSpanIDToAttrs(ctx, attrs)
	attrs = setUserIDToAttrs(ctx, attrs)
	attrs = hide.Attrs(attrs)
	r.AddAttrs(attrs...)

	logger.Handler().Handle(ctx, r)
}

func Info(ctx context.Context, msg string, args ...slog.Attr) {
	Log(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...slog.Attr) {
	Log(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, err error, args ...slog.Attr) {
	errWrap := errors.WrapSkip(ctx, err, 1)
	if errWrap == nil {
		return
	}
	errStack := errWrap.(*errors.Error)
	args = append(args, utils.KeyStacktrace(errStack.ErrorStack()))
	args = append(args, utils.KeyError(errStack))
	args = append(args, errStack.Attrs()...)

	Log(ctx, slog.LevelError, err.Error(), args...)
}

func Debug(ctx context.Context, msg string, args ...slog.Attr) {
	Log(ctx, slog.LevelDebug, msg, args...)
}

func Fatal(ctx context.Context, err error, args ...slog.Attr) {
	Error(ctx, err, args...)
	os.Exit(1)
}

func Recovery(ctx context.Context, err *error, attrs ...slog.Attr) {
	if rr := recover(); rr != nil {
		panicMsg := fmt.Sprint(rr)
		if err != nil {
			*err = errors.New(ctx, panicMsg)
		}
		pc, _, _, _ := runtime.Caller(2)
		caller := runtime.FuncForPC(pc).Name()
		panicErr := errors.WrapPrefix(ctx,
			errors.New(ctx, panicMsg),
			utils.GetOpName(caller),
		)
		attrs = append(attrs, utils.KeyPanic(panicMsg, 2))
		Error(ctx, panicErr, attrs...)
	}
}

func setUserIDToAttrs(ctx context.Context, attrs []slog.Attr) []slog.Attr {
	if userID, ok := ctx.Value(utils.UserIDKey).(string); ok {
		attrs = append(attrs, slog.String(string(utils.UserIDKey), userID))
	}
	return attrs
}
