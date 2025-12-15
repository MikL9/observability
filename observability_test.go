package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MikL9/observability/logger/errors"
)

var constError = errors.NewConstError("const error")

func nameForTest(ctx context.Context) (err error) {
	defer WrapError(ctx, &err)
	return constError
}

func TestWrapPrefixError(t *testing.T) {
	var err error
	ctx := context.Background()

	err = func() error {
		defer WrapError(ctx, nil)
		return nil
	}()
	require.NoError(t, err)

	err = func() (err error) {
		defer WrapError(ctx, &err)
		return err
	}()
	require.NoError(t, err)

	err = nameForTest(ctx)
	require.Error(t, err)
	assert.Equal(t, err.Error(), "name for test: const error")
	errObj, ok := err.(*errors.Error)
	assert.True(t, ok, "failed to convert observability Error object")
	assert.NotEmpty(t, errObj.ErrorStack())
}
