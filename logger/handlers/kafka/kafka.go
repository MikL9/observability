package kafka

import (
	"context"
	"encoding/json"
	"time"

	"log/slog"

	slogcommon "github.com/samber/slog-common"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Converter func(loggerAttr []slog.Attr, record *slog.Record) map[string]any

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// Kafka Writer
	Client KafkaProducer

	Converter    Converter
	Marshaler    func(v any) ([]byte, error)
	ReplaceAttr  func(groups []string, a slog.Attr) slog.Attr
	DefaultAttrs []slog.Attr

	Timeout time.Duration // default: 10s

	AddSource bool
}

func (o Option) NewHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.Client == nil {
		panic("missing Kafka client")
	}

	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}

	if o.Converter == nil {
		o.Converter = DefaultConverter
	}

	if o.Marshaler == nil {
		o.Marshaler = json.Marshal
	}

	if o.DefaultAttrs == nil {
		o.DefaultAttrs = []slog.Attr{}
	}

	return &Handler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

type Handler struct {
	attrs  []slog.Attr
	groups []string
	option Option
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	payload := h.option.Converter(
		h.option.DefaultAttrs,
		&record,
	)

	return h.publish(ctx, record.Time, payload)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		option: h.option,
		attrs:  slogcommon.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return h
	}

	return &Handler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

func (h *Handler) publish(ctx context.Context, timestamp time.Time, payload map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, h.option.Timeout)
	defer cancel()
	key := []byte(timestamp.Format(time.RFC3339))

	// bearer:disable go_lang_deserialization_of_user_input
	values, err := h.option.Marshaler(payload)
	if err != nil {
		return err
	}

	err = h.option.Client.WriteMessages(
		ctx,
		kafka.Message{
			Key:   key,
			Value: values,
		},
	)
	return err
}
