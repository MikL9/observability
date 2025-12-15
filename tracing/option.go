package tracing

import "time"

type traceProviderOptions struct {
	env string
	// sampleRate определяет частоту выборки трассировок.
	// Значение больше или равное 1 выберет все трассировки,
	// значение 0.5 выберет 50% трассировок, а значение 0 отключит выборку.
	sampleRate float64
	// maxExportBatchSize определяет максимальный размер батча для экспорта трассировок.
	maxExportBatchSize int
	// batchTimeout определяет максимальное время ожидания перед отправкой данных (пакета) экспортеру трассировок.
	// Если прошло более batchTimeout с момента последней отправки данных, текущий пакет будет немедленно отправлен.
	// Время указывается в миллисекундах.
	batchTimeout time.Duration
}

// TraceProviderOption определяет функцию для установки параметров конфигурации трассировки.
type TraceProviderOption func(*traceProviderOptions)

// WithSampleRate устанавливает частоту выборки трассировок.
func WithSampleRate(sampleRate float64) TraceProviderOption {
	return func(opts *traceProviderOptions) {
		opts.sampleRate = sampleRate
	}
}

// WithMaxExportBatchSize устанавливает максимальный размер пакета для экспорта трассировок.
func WithMaxExportBatchSize(maxExportBatchSize int) TraceProviderOption {
	return func(opts *traceProviderOptions) {
		opts.maxExportBatchSize = maxExportBatchSize
	}
}

// WithBatchTimeout устанавливает максимальное время ожидания перед отправкой данных (пакета) экспортеру трассировок.
func WithBatchTimeout(batchTimeout time.Duration) TraceProviderOption {
	return func(opts *traceProviderOptions) {
		opts.batchTimeout = batchTimeout
	}
}

// WithEnvAttribute добавляет к каждому трейсу атрибут env
func WithEnvAttribute(env string) TraceProviderOption {
	return func(opts *traceProviderOptions) {
		opts.env = env
	}
}
