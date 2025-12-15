package pretty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/MikL9/observability/utils"
)

const (
	timeFormat = "[15:04:05.000]"

	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

type Handler struct {
	h slog.Handler
	b *bytes.Buffer
	m *sync.Mutex
}

func NewHandler(opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}
	return &Handler{
		b: b,
		h: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		m: &sync.Mutex{},
	}
}

func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func compileAttrs(attrs map[string]any, ident int) string {
	msg := strings.Builder{}
	for key, val := range attrs {
		if key == utils.StacktraceKey {
			continue
		}

		msg.WriteString(strings.Repeat(" ", ident))
		if inherit, ok := val.(map[string]any); ok {
			msg.WriteString(fmt.Sprintf("%s=\n", key))
			msg.WriteString(compileAttrs(inherit, ident*2))
			continue
		}
		msg.WriteString(fmt.Sprintf("%s=%#v\n", key, val))
	}
	return msg.String()
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {

	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	msg := strings.Builder{}
	msg.WriteString(colorize(lightGray, r.Time.Format(timeFormat)))
	msg.WriteRune(' ')
	msg.WriteString(level)
	msg.WriteRune(' ')
	msg.WriteString(colorize(white, r.Message))
	msg.WriteString(" {\n")

	msg.WriteString(compileAttrs(attrs, 2))

	if val, ok := attrs[utils.StacktraceKey]; ok {
		msg.WriteString(fmt.Sprintf("    %s=\n%s", colorize(red, utils.StacktraceKey), val))
	}
	msg.WriteString("}\n")

	fmt.Print(msg.String())

	return nil
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{h: h.h.WithAttrs(attrs), b: h.b, m: h.m}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{h: h.h.WithGroup(name), b: h.b, m: h.m}
}
