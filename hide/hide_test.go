package hide

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setTestConvertor() {
	SetDefaultConverter(NewConverter(
		WithMaskNameRule([]string{"lastname"}),
		WithMaskEmailRule([]string{"email"}),
		WithFullExcludeRule([]string{"token", "password"}),
	))
}

func TestHideAttrs(t *testing.T) {
	setTestConvertor()
	testCases := []struct {
		name     string
		input    []slog.Attr
		expected []slog.Attr
	}{
		{
			"empty slice",
			[]slog.Attr{},
			[]slog.Attr{},
		},
		{
			"slice without replace value",
			[]slog.Attr{slog.Int("id", 2), slog.Bool("error", true)},
			[]slog.Attr{slog.Int("id", 2), slog.Bool("error", true)},
		},
		{
			"slice with hide and mask values",
			[]slog.Attr{slog.String("token", "whefF63o5tA"), slog.String("password", "Sa&24rjfso")},
			[]slog.Attr{slog.String("token", "***********"), slog.String("password", "**********")},
		},
		{
			"slice with hide and mask empty values",
			[]slog.Attr{slog.String("token", ""), slog.String("password", "")},
			[]slog.Attr{slog.String("token", ""), slog.String("password", "")},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			Attrs(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestHideByKeyValue(t *testing.T) {
	setTestConvertor()
	testCases := []struct {
		name        string
		inputKey    string
		inputVal    string
		expectedVal string
	}{
		{"empty key val", "", "", ""},
		{"empty key", "", "load", "load"},
		{"empty val", "username", "", ""},
		{"hide val", "password", "secret", "******"},
		{"empty hide val", "password", "", ""},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			outputVal := Hide(tt.inputKey, tt.inputVal)
			assert.Equal(t, tt.expectedVal, outputVal)
		})
	}
}
