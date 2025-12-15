package hide

import (
	"log/slog"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const input = `
{
 "username": "employee",
 "email": "employer@now.com",
 "id": 2,
 "age": null,
 "cvc": 123,
 "password": "awesome",
 "bio": {"lastname": "Last"}
}
`

const partialInput = `
{
 "username": "employee",
 "email": "employer@now.com",
 "id": 2,
 "age": null,
 "cvc": 123,
 "password": "awesome",
 "bio": {"la
}
`

func TestMaskSensitiveJSONFields(t *testing.T) {
	setTestConvertor()
	data, err := MaskSensitiveJSONFields([]byte(input))
	require.NoError(t, err)
	assert.Equal(t,
		`{"username":"employee","email":"emp*****@now.com","id":2,"age":null,"cvc":123,"password":"*******","bio":{"lastname":"La**"}}`,
		data,
	)
}

func TestWrongType(t *testing.T) {
	setTestConvertor()
	data := JSON("name", []byte(`<xml version=1.0>\n<tag>close</tag>`), 10_000, true)
	assert.Equal(t,
		`<xml version=1.0>\n<tag>close</tag>`,
		data.Value.String(),
	)
}

func TestPartialMaskSensitiveJSONFields(t *testing.T) {
	setTestConvertor()
	data, err := MaskSensitiveJSONFields([]byte(partialInput))

	require.Error(t, err)
	assert.Equal(t,
		`{"username":"employee","email":"emp*****@now.com","id":2,"age":null,"cvc":123,"password":"*******","bio":{`,
		data,
	)
}

func TestLimitJSONField(t *testing.T) {
	setTestConvertor()
	attr := JSON("json", []byte(input), 100, false)

	assert.Equal(t, "json_text", attr.Key)
	assert.Equal(t, slog.KindString, attr.Value.Kind())
	assert.Equal(t,
		`{"username":"employee","email":"emp*****@now.com","id":2,"age":null,"cvc":123`,
		attr.Value.String(),
	)
}

func TestJSONField(t *testing.T) {
	setTestConvertor()
	attr := JSON("json", []byte(input), 10_000, false)

	assert.Equal(t, "json", attr.Key)
	assert.Equal(t, slog.KindGroup, attr.Value.Kind())
	sl := attr.Value.Group()
	assert.Equal(t, 7, len(sl))
	expectedData := []slog.Attr{
		slog.Float64("cvc", 123),
		slog.String("password", "*******"),
		slog.Any("bio", map[string]any{"lastname": "La**"}),
		slog.String("username", "employee"),
		slog.String("email", "emp*****@now.com"),
		slog.Float64("id", 2),
		slog.Any("age", nil),
	}
	sort.Slice(expectedData, func(i, j int) bool {
		return expectedData[i].Key > expectedData[j].Key
	})
	sort.Slice(sl, func(i, j int) bool {
		return sl[i].Key > sl[j].Key
	})

	for i, data := range expectedData {
		// hack из-за невозможности корректно сравнить группу в группе
		if data.Key == "bio" {
			assert.Equal(t, data.Value, sl[i].Value)
			continue
		}
		assert.True(t, data.Equal(sl[i]))
	}
}

func TestJSONStringField(t *testing.T) {
	setTestConvertor()
	attr := JSON("json", []byte(input), 10_000, true)

	assert.Equal(t, "json_text", attr.Key)
	assert.Equal(t, slog.KindString, attr.Value.Kind())
	assert.Equal(t,
		`{"username":"employee","email":"emp*****@now.com","id":2,"age":null,"cvc":123,"password":"*******","bio":{"lastname":"La**"}}`,
		attr.Value.String(),
	)
}
