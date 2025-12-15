package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	sentry2 "github.com/getsentry/sentry-go"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/MikL9/observability/logger/errors"
	kafkaHandler "github.com/MikL9/observability/logger/handlers/kafka"
	"github.com/MikL9/observability/logger/handlers/sentry"
	storage2 "github.com/MikL9/observability/logger/handlers/storage"
	"github.com/MikL9/observability/storage"
	"github.com/MikL9/observability/utils"
)

func keyUserID(v string) slog.Attr        { return slog.String(string(utils.UserIDKey), v) }
func keyIdentificationID(v int) slog.Attr { return slog.Int("identification_id", v) }

func keyID(v int) slog.Attr                         { return slog.Int("id", v) }
func keyStatus(v string) slog.Attr                  { return slog.String("status", v) }
func keyRequest(v slog.LogValuer) slog.Attr         { return Object("request", v) }
func keyRequests[T slog.LogValuer](v []T) slog.Attr { return Array("requests", v) }

var requestLogMsg = map[string]any{"id": float64(1), "status": "canceled"}

type request struct {
	ID     int
	Status string
}

func (r *request) LogValue() slog.Value {
	return slog.GroupValue(
		keyID(r.ID),
		keyStatus(r.Status),
	)
}

// newTestRequest создание структуры для проверки сериализации через Log() интерфейс
func newTestRequest() *request {
	return &request{ID: 1, Status: "canceled"}
}

const staticTime = "1999-12-31T23:59:57Z"

// testStaticTime мок для проверки времени через time.Now
func testStaticTime(t *testing.T) {
	timeNow = func() time.Time {
		now, err := time.Parse(time.RFC3339, staticTime)
		require.NoError(t, err)
		return now
	}
	t.Cleanup(func() {
		timeNow = time.Now
	})
}

type stdoutWriterMock struct {
	input chan []byte
}

// newStdoutWriter мок для os.Stdout
func newStdoutWriter() *stdoutWriterMock {
	return &stdoutWriterMock{
		input: make(chan []byte, 100),
	}
}

func (s *stdoutWriterMock) Write(w []byte) (int, error) {
	s.input <- w
	return 0, nil
}

func (s *stdoutWriterMock) Read() []byte {
	return <-s.input
}

func assertRecord(t *testing.T, output []byte, expected map[string]any) {
	var logMessage map[string]any
	if err := json.Unmarshal(output, &logMessage); err != nil {
		require.NoError(t, err)
	}

	assert.Equal(t, logMessage, expected)
}

func TestLogInfo(t *testing.T) {
	testStaticTime(t)
	ctx, cancel := context.WithCancel(context.Background())
	// закрытый контекст не должен влиять на работу логгера
	cancel()

	writer := newStdoutWriter()
	err := SetupLogger(storage2.NewHandler(slog.NewJSONHandler(writer, nil)))
	require.NoError(t, err)

	ctx = storage.SetContextAttr(ctx,
		keyUserID("5"),
		keyRequest(newTestRequest()),
		keyRequests([]*request{newTestRequest(), newTestRequest()}),
	)
	Info(ctx, "activate user", keyIdentificationID(5432))

	assertRecord(t, writer.Read(),
		map[string]any{
			"identification_id": float64(5432),
			"level":             slog.LevelInfo.String(),
			"msg":               "activate user",
			"time":              staticTime,
			"user_id":           "5",
			"request":           requestLogMsg,
			"requests":          []any{requestLogMsg, requestLogMsg},
		},
	)
}

func TestLogFormat(t *testing.T) {
	testStaticTime(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := newStdoutWriter()

	// Добавляем log_format в handler
	handler := slog.NewJSONHandler(writer, nil).WithAttrs([]slog.Attr{
		slog.String("log_format", "json_test"),
	})
	err := SetupLogger(storage2.NewHandler(handler))
	require.NoError(t, err)

	ctx = storage.SetContextAttr(ctx,
		keyUserID("5"),
		keyRequest(newTestRequest()),
		keyRequests([]*request{newTestRequest(), newTestRequest()}),
	)
	Info(ctx, "activate user", keyIdentificationID(5432))

	writerRes := writer.Read()
	assertRecord(t, writerRes,
		map[string]any{
			"identification_id": float64(5432),
			"level":             slog.LevelInfo.String(),
			"msg":               "activate user",
			"time":              staticTime,
			"user_id":           "5",
			"request":           requestLogMsg,
			"requests":          []any{requestLogMsg, requestLogMsg},
			"log_format":        "json_test",
		},
	)
}

func TestLogWarn(t *testing.T) {
	testStaticTime(t)
	ctx, cancel := context.WithCancel(context.Background())
	// закрытый контекст не должен влиять на работу логгера
	cancel()

	writer := newStdoutWriter()
	err := SetupLogger(storage2.NewHandler(slog.NewJSONHandler(writer, nil)))
	require.NoError(t, err)

	Warn(ctx, "user already exists", keyIdentificationID(5432))

	assertRecord(t, writer.Read(),
		map[string]any{
			"level":             slog.LevelWarn.String(),
			"msg":               "user already exists",
			"time":              staticTime,
			"identification_id": float64(5432),
		})
}

func TestLogError(t *testing.T) {
	testStaticTime(t)
	ctx, cancel := context.WithCancel(context.Background())
	// закрытый контекст не должен влиять на работу логгера
	cancel()

	err := errors.New(ctx, "syntax error",
		keyUserID("5"),
		keyRequest(newTestRequest()),
		keyRequests([]*request{newTestRequest(), newTestRequest()}),
	)

	writer := newStdoutWriter()
	require.NoError(t,
		SetupLogger(storage2.NewHandler(slog.NewJSONHandler(writer, nil))),
	)

	Error(ctx, err, keyIdentificationID(5432))

	assertRecord(t, writer.Read(),
		map[string]any{
			"time":              staticTime,
			"level":             slog.LevelError.String(),
			"msg":               "syntax error",
			"error_msg":         "syntax error",
			"user_id":           "5",
			"request":           requestLogMsg,
			"requests":          []any{requestLogMsg, requestLogMsg},
			"identification_id": float64(5432),
			"stacktrace":        err.(*errors.Error).ErrorStack(),
		},
	)
}

func TriggerPanicFunc(ctx context.Context) (err error) {
	defer Recovery(ctx, &err, keyIdentificationID(5432))
	panic("nil pointer dereference")
}

func TestRecovery(t *testing.T) {
	testStaticTime(t)
	ctx := context.Background()

	writer := newStdoutWriter()
	err := SetupLogger(storage2.NewHandler(slog.NewJSONHandler(writer, nil)))
	require.NoError(t, err)

	err = TriggerPanicFunc(ctx)

	var logMessage map[string]any
	if err := json.Unmarshal(writer.Read(), &logMessage); err != nil {
		require.NoError(t, err)
	}

	require.Error(t, err)
	assert.Equal(t, err.Error(), "nil pointer dereference")

	assert.Equal(t, staticTime, logMessage["time"])
	assert.Equal(t, "ERROR", logMessage["level"])
	assert.Equal(t, float64(5432), logMessage["identification_id"])

	panicMsg, ok := logMessage["panic"].(map[string]any)
	assert.True(t, ok, "panic attr must be a map")
	assert.Contains(t, panicMsg["file"], "logger/logger_test.go")
	assert.Contains(t, panicMsg["function"], "TriggerPanicFunc")
	assert.Equal(t, "nil pointer dereference", panicMsg["recover-error"])
}

func TestRecoveryNilError(t *testing.T) {
	func() {
		defer Recovery(context.Background(), nil)
		panic("happened")
	}()
}

func TestRecoveryWithoutPanic(t *testing.T) {
	Recovery(context.Background(), nil)

	func() {
		defer Recovery(context.Background(), nil)
	}()
}

func TestSentryHandler(t *testing.T) {
	var outputEvent *sentry2.Event
	testStaticTime(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := SetupLogger(sentry.NewHandler(slog.LevelError, "0.21", "prod"))
	require.NoError(t, err)

	sentry2.Init(sentry2.ClientOptions{
		BeforeSend: func(event *sentry2.Event, hint *sentry2.EventHint) *sentry2.Event {
			outputEvent = event
			return event
		},
	})

	Debug(ctx, "debug msg")
	Info(ctx, "info msg")
	Warn(ctx, "warn msg")
	require.Nil(t, outputEvent, "Сообщения ниже уровня ERROR попадают в sentry")

	ctx = storage.SetContextAttr(ctx, keyUserID("18552"), keyRequest(newTestRequest()))
	err = errors.New(ctx, "error msg", keyIdentificationID(5432))
	Error(ctx, err)
	require.NotNil(t, outputEvent, "cообщение уровня ERROR не попало в sentry")
	assert.Equal(t, sentry2.LevelError, outputEvent.Level)
	assert.Equal(t, staticTime, outputEvent.Timestamp.Format(time.RFC3339))
	assert.Equal(t, "0.21", outputEvent.Release)
	assert.Equal(t, "prod", outputEvent.Environment)
	assert.Equal(t, "18552", outputEvent.User.ID)
	assert.Equal(t, 1, len(outputEvent.Exception))
	assert.Equal(t, "go", outputEvent.Exception[0].Mechanism.Type)
	assert.Equal(t, true, *outputEvent.Exception[0].Mechanism.Handled)
	assert.Equal(t, "error msg", outputEvent.Exception[0].Type)
	assert.Equal(t, "*errors.errorString error msg", outputEvent.Exception[0].Value)
	assert.Equal(t, err.(*errors.Error).SentryStackTrace(), outputEvent.Exception[0].Stacktrace)

	assert.Equal(t, map[string]any{
		"identification_id": "5432",
		"request":           "[id=1 status=canceled]",
	}, outputEvent.Contexts["extra"])
}

type MockKafkaClient struct {
	result []kafka.Message
}

func (m *MockKafkaClient) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.result = append(m.result, msgs...)
	return nil
}

func TestKafkaLogging(t *testing.T) {
	testStaticTime(t)
	ctx := context.Background()
	kafkaMock := &MockKafkaClient{}
	expectedErr := errors.New(ctx, "syntax error", keyIdentificationID(5432))

	slog.SetDefault(
		slog.New(
			kafkaHandler.Option{Client: kafkaMock}.NewHandler(),
		),
	)
	Error(
		ctx,
		expectedErr,
		keyUserID("5"),
		keyRequest(newTestRequest()),
		keyRequests([]*request{newTestRequest(), newTestRequest()}),
	)

	res := kafkaMock.result
	assert.Equal(t, 1, len(res))

	key := string(res[0].Key)
	assert.Equal(t, staticTime, key)

	assertRecord(t, res[0].Value,
		map[string]any{
			"time":              staticTime,
			"level":             slog.LevelError.String(),
			"msg":               "syntax error",
			"user_id":           "5",
			"request":           requestLogMsg,
			"requests":          []any{requestLogMsg, requestLogMsg},
			"identification_id": float64(5432),
			"stacktrace":        expectedErr.(*errors.Error).ErrorStack(),
		},
	)
}
