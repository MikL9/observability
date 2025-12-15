package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type person struct {
	Name string
	Age  int
}

func (p person) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", p.Name),
		slog.Int("age", p.Age),
	)
}

func TestObject(t *testing.T) {
	p := person{"John", 20}
	group := Object("person", p)
	assert.Equal(t, slog.KindGroup, group.Value.Kind())
	expectedAttrs := []slog.Attr{
		slog.String("name", "John"),
		slog.Int("age", 20),
	}
	for i, attr := range group.Value.Group() {
		assert.Equal(t, expectedAttrs[i], attr)
	}
}

func TestNilObject(t *testing.T) {
	var p *person
	group := Object("person", p)

	assert.Equal(t, "person", group.Key)
	assert.Equal(t, slog.KindAny, group.Value.Kind())
	assert.Equal(t, nil, group.Value.Any())
}

func TestArray(t *testing.T) {
	persons := []person{{"John", 20}, {"Bobr", 3}}
	group := Array("persons", persons)

	assert.Equal(t, group.Key, "persons")
	assert.Equal(t, slog.KindAny, group.Value.Kind())
	assert.Equal(t,
		[]map[string]any{{"name": "John", "age": int64(20)}, {"name": "Bobr", "age": int64(3)}},
		group.Value.Any(),
	)
}

func TestNilArray(t *testing.T) {
	var persons []*person
	group := Array("persons", persons)

	assert.Equal(t, group.Key, "persons")
	assert.Equal(t, slog.KindAny, group.Value.Kind())
	assert.Equal(t, []map[string]any{}, group.Value.Any())

	persons = append(persons, nil)

	assert.Equal(t, group.Key, "persons")
	assert.Equal(t, slog.KindAny, group.Value.Kind())
	assert.Equal(t, []map[string]any{}, group.Value.Any())
}
