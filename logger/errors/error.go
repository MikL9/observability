package errors

import (
	"bytes"
	"fmt"
	"go/build"
	"log/slog"
	"reflect"
	"slices"
	"strings"

	"github.com/getsentry/sentry-go"
)

var goRoot = strings.ReplaceAll(build.Default.GOROOT, "\\", "/")

// MaxStackDepth The maximum number of stackframes on any error.
var MaxStackDepth = 50

type ConstError struct {
	msg string
}

func (c ConstError) Error() string {
	return c.msg
}

// Error is an error with an attached stacktrace. It can be used
// wherever the builtin error interface is expected.
type Error struct {
	Err    error
	stack  []uintptr
	frames []StackFrame
	prefix string
	attrs  []slog.Attr
}

// Error returns the underlying error's message.
func (err *Error) Error() string {

	msg := err.Err.Error()
	if err.prefix != "" {
		msg = fmt.Sprintf("%s: %s", err.prefix, msg)
	}

	return msg
}

func (err *Error) Attrs() []slog.Attr {
	return err.attrs
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (err *Error) Stack() []byte {
	buf := bytes.Buffer{}

	for _, frame := range err.StackFrames() {
		buf.WriteString(frame.String())
	}

	return buf.Bytes()
}

func (err *Error) Callers() []uintptr {
	return err.stack
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (err *Error) ErrorStack() string {
	return string(err.Stack())
}

// StackFrames returns an array of frames containing information about the stack
func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))

		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}

	return err.frames
}

func (err *Error) SentryStackTrace() *sentry.Stacktrace {
	if err.frames == nil {
		err.StackFrames()
	}
	stackFrames := make([]sentry.Frame, 0, len(err.frames))
	for _, frame := range err.StackFrames() {
		preContext, contextLine, postContext, _ := frame.sourceLines()
		sentryFrame := sentry.Frame{
			Filename:    frame.File,
			Module:      frame.Package,
			Lineno:      frame.LineNumber,
			Function:    frame.Name,
			PreContext:  preContext,
			ContextLine: contextLine,
			PostContext: postContext,
		}
		if !strings.HasPrefix(sentryFrame.Filename, goRoot) &&
			!strings.Contains(sentryFrame.Module, "vendor") &&
			!strings.Contains(sentryFrame.Module, "third_party") {
			sentryFrame.InApp = true
		}
		stackFrames = append(stackFrames, sentryFrame)
	}
	slices.Reverse(stackFrames)
	return &sentry.Stacktrace{Frames: stackFrames}
}

// TypeName returns the type this error. e.g. *errors.stringError
func (err *Error) TypeName() string {
	return reflect.TypeOf(err.Err).String()
}

// Unwrap return the wrapped error (implements api for As function)
func (err *Error) Unwrap() error {
	return err.Err
}
