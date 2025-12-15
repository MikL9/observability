package storage

import (
	"context"
	"log/slog"

	"github.com/MikL9/observability/hide"
	"github.com/MikL9/observability/storage"
)

type Handler struct {
	next slog.Handler
}

func NewHandler(next slog.Handler) *Handler {
	return &Handler{next: next}
}

func (h *Handler) Enabled(ctx context.Context, rec slog.Level) bool {
	return h.next.Enabled(ctx, rec)
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	rec.AddAttrs(storage.ExportSlogAttributes(ctx)...)
	return h.next.Handle(ctx, rec)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hide.Attrs(attrs)
	return &Handler{next: h.next.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{next: h.next.WithGroup(name)}
}
