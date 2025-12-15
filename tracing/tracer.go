package tracing

import (
	"context"
	"strings"

	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type SpanWrapper struct {
	ctx *context.Context
	trace.Span
	op string
}

func New(ctx *context.Context, op string) *SpanWrapper {
	return &SpanWrapper{
		ctx: ctx,
		op:  op,
	}
}

func (s *SpanWrapper) Start() {
	var ctx context.Context
	ctx, s.Span = otel.Tracer(s.op).Start(*s.ctx, s.op)
	*s.ctx = ctx
}

func (s *SpanWrapper) StopWrap(e *error) {
	err := *e
	s.SetAttributes(storage.ExportOtelAttributes(*s.ctx)...)
	if err != nil {
		opSeparateCase := strings.Join(strings.Split(s.op, "_"), " ")
		*e = errors.WrapPrefix(*s.ctx, err, opSeparateCase)
		s.RecordError(err)
		s.SetStatus(codes.Error, err.Error())
	}
	s.End()
}
