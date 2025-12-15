package logger

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	kafkaHandler "github.com/MikL9/observability/logger/handlers/kafka"
	"github.com/MikL9/observability/logger/handlers/pretty"
	sentryHandler "github.com/MikL9/observability/logger/handlers/sentry"
)

const (
	envDev   = "dev"
	envLocal = "local"
	envProd  = "prod"
	envStage = "stage"
)

type Option struct {
	Handlers []slog.Handler
}

func WithDefaultHandler(env string) slog.Handler {
	switch env {
	case envLocal:
		return pretty.NewHandler(&slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	case envDev, envStage:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	case envProd:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	default:
		panic("Unknown env " + env)
	}
}

func WithKafkaHandler(option *kafkaHandler.Option) slog.Handler {
	return option.NewHandler()
}

func WithSentryHandler(level slog.Leveler, dsn, release, env string) slog.Handler {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:           dsn,
		EnableTracing: false,
		HTTPTransport: sentryHandler.NewTransport(http.DefaultTransport),
	}); err != nil {
		panic(err)
	}
	defer sentry.Flush(2 * time.Second)
	return sentryHandler.NewHandler(level, release, env)
}
