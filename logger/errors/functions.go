package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/MikL9/observability/storage"
)

func combineAttrs(ctx context.Context, attrs []slog.Attr) []slog.Attr {
	return append(attrs, storage.ExportSlogAttributes(ctx)...)
}

func NewConstError(msg string) error {
	return ConstError{
		msg: msg,
	}
}

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(ctx context.Context, s string, attrs ...slog.Attr) error {
	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])
	return &Error{
		Err:   fmt.Errorf("%s", s),
		stack: stack[:length],
		attrs: combineAttrs(ctx, attrs),
	}
}

func wrap(ctx context.Context, e error, skip int, attrs ...slog.Attr) error {
	if e == nil {
		return nil
	}
	if err, ok := e.(*Error); ok {
		err.attrs = combineAttrs(ctx, append(err.attrs, attrs...))
		return err
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return &Error{
		Err:   e,
		stack: stack[:length],
		attrs: combineAttrs(ctx, attrs),
	}
}

func Wrap(ctx context.Context, e error, attrs ...slog.Attr) error {
	return wrap(ctx, e, 1, attrs...)
}

func WrapSkip(ctx context.Context, e error, skip int, attrs ...slog.Attr) error {
	return wrap(ctx, e, skip, attrs...)
}

// WrapPrefix makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The prefix parameter is used to add a prefix to the
// error message when calling Error(). The skip parameter indicates how far
// up the stack to start the stacktrace. 0 is from the current call,
// 1 from its caller, etc.
func WrapPrefix(ctx context.Context, e error, prefix string, attrs ...slog.Attr) error {
	if e == nil {
		return nil
	}

	err := wrap(ctx, e, 1).(*Error)
	if err.prefix != "" {
		prefix = fmt.Sprintf("%s: %s", prefix, err.prefix)
	}

	err.attrs = append(err.attrs, attrs...)

	return &Error{
		Err:    err.Err,
		stack:  err.stack,
		prefix: prefix,
		attrs:  err.attrs,
	}
}

func WrapPrefixSkip(ctx context.Context, e error, prefix string, skip int, attrs ...slog.Attr) error {
	if e == nil {
		return nil
	}

	err := wrap(ctx, e, 1+skip).(*Error)
	if err.prefix != "" {
		prefix = fmt.Sprintf("%s: %s", prefix, err.prefix)
	}

	err.attrs = append(err.attrs, attrs...)

	return &Error{
		Err:    err.Err,
		stack:  err.stack,
		prefix: prefix,
		attrs:  err.attrs,
	}
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
