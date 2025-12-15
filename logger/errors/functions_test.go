package errors

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type customError struct {
	error
}

func TestErrorIs(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name string
		err  error
	}{
		{"fmt error", fmt.Errorf("fmt some error")},
		{"custom error", customError{}},
		{"const error", NewConstError("const some error")},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, Is(tt.err, tt.err), "original error mismatch")
			assert.True(t, Is(Wrap(ctx, tt.err), tt.err), "wrap mismatch")
			assert.True(t, Is(WrapPrefix(ctx, tt.err, "prefix"), tt.err), "wrap prefix mismatch")
			assert.True(t, Is(WrapSkip(ctx, tt.err, 1), tt.err), "wrap skip mismatch")
			assert.True(t, Is(WrapPrefixSkip(ctx, tt.err, "prefix", 1), tt.err), "wrap prefix skip mismatch")
		})
	}
}

func TestWrapPrefix(t *testing.T) {
	ctx := context.Background()
	originalErr := NewConstError("some error")
	err := WrapPrefix(ctx, originalErr, "prefix")
	assert.Equal(t, "prefix: some error", err.Error())
	err = WrapPrefix(ctx, err, "another prefix")
	assert.Equal(t, "another prefix: prefix: some error", err.Error())
}

func TestAttrsErrors(t *testing.T) {
	var err error
	ctx := context.Background()
	attrs := []slog.Attr{slog.String("user_id", "18552"), slog.Int("count", 5)}
	err = New(ctx, "new error", attrs...)
	assert.Equal(t,
		[]slog.Attr{
			slog.String("user_id", "18552"),
			slog.Int("count", 5),
		},
		err.(*Error).Attrs(),
	)

	err = Wrap(ctx, err, slog.Bool("created", true))
	assert.Equal(t,
		[]slog.Attr{
			slog.String("user_id", "18552"),
			slog.Int("count", 5),
			slog.Bool("created", true),
		},
		err.(*Error).Attrs(),
	)

	err = WrapPrefix(ctx, err, "some staff", slog.Int("work", 5))
	assert.Equal(t,
		[]slog.Attr{
			slog.String("user_id", "18552"),
			slog.Int("count", 5),
			slog.Bool("created", true),
			slog.Int("work", 5),
		},
		err.(*Error).Attrs(),
	)
}
