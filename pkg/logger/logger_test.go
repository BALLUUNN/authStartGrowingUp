package logger

import (
	"strconv"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestConfigZapConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Config{
		ServiceName: "auth-service",
	}.ZapConfig()
	if err != nil {
		t.Fatalf("ZapConfig() error = %v", err)
	}

	if cfg.Encoding != string(Console) {
		t.Fatalf("Encoding = %q, want %q", cfg.Encoding, Console)
	}

	if got := cfg.Level.Level(); got != zap.DebugLevel {
		t.Fatalf("Level = %s, want %s", got, zap.DebugLevel)
	}

	if cfg.InitialFields[ServiceKey] != "auth-service" {
		t.Fatalf("service initial field = %v", cfg.InitialFields[ServiceKey])
	}

	if cfg.InitialFields[EnvironmentKey] != string(Development) {
		t.Fatalf("environment initial field = %v", cfg.InitialFields[EnvironmentKey])
	}

	if len(cfg.OutputPaths) != 1 || cfg.OutputPaths[0] != "stdout" {
		t.Fatalf("OutputPaths = %v", cfg.OutputPaths)
	}

	if len(cfg.ErrorOutputPaths) != 1 || cfg.ErrorOutputPaths[0] != "stderr" {
		t.Fatalf("ErrorOutputPaths = %v", cfg.ErrorOutputPaths)
	}
}

func TestConfigZapConfigRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	_, err := Config{
		ServiceName: "auth-service",
		Level:       "verbose",
	}.ZapConfig()
	if err == nil {
		t.Fatal("ZapConfig() error = nil, want error")
	}

	_, err = Config{
		ServiceName: "auth-service",
		Format:      "pretty",
	}.ZapConfig()
	if err == nil {
		t.Fatal("ZapConfig() format error = nil, want error")
	}

	_, err = Config{
		ServiceName: "auth-service",
		Sampling: &SamplingConfig{
			Initial:    0,
			Thereafter: 1,
		},
	}.ZapConfig()
	if err == nil {
		t.Fatal("ZapConfig() sampling error = nil, want error")
	}
}

func TestConfigZapConfigNormalizesExplicitValues(t *testing.T) {
	t.Parallel()

	cfg, err := Config{
		ServiceName:      " auth-service ",
		OutputPaths:      []string{"  ", " stdout ", "\t./logs/app.log\t"},
		ErrorOutputPaths: []string{"", "\nstderr\n"},
	}.ZapConfig()
	if err != nil {
		t.Fatalf("ZapConfig() error = %v", err)
	}

	if cfg.InitialFields[ServiceKey] != "auth-service" {
		t.Fatalf("service initial field = %v", cfg.InitialFields[ServiceKey])
	}

	if len(cfg.OutputPaths) != 2 || cfg.OutputPaths[0] != "stdout" || cfg.OutputPaths[1] != "./logs/app.log" {
		t.Fatalf("OutputPaths = %v", cfg.OutputPaths)
	}

	if len(cfg.ErrorOutputPaths) != 1 || cfg.ErrorOutputPaths[0] != "stderr" {
		t.Fatalf("ErrorOutputPaths = %v", cfg.ErrorOutputPaths)
	}
}

func TestLoggerWithStructuredContext(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.InfoLevel)
	logger := FromZap(zap.New(core)).
		WithRequestID("req-123").
		WithActorID("user-7").
		WithAction("token.issue").
		WithResult("success")

	logger.Info("token issued", String("scope", "access"))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("entries count = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Message != "token issued" {
		t.Fatalf("message = %q", entry.Message)
	}

	ctx := entry.ContextMap()
	if ctx[RequestIDKey] != "req-123" {
		t.Fatalf("request_id = %v", ctx[RequestIDKey])
	}
	if ctx[ActorIDKey] != "user-7" {
		t.Fatalf("actor_id = %v", ctx[ActorIDKey])
	}
	if ctx[ActionKey] != "token.issue" {
		t.Fatalf("action = %v", ctx[ActionKey])
	}
	if ctx[ResultKey] != "success" {
		t.Fatalf("result = %v", ctx[ResultKey])
	}
	if ctx["scope"] != "access" {
		t.Fatalf("scope = %v", ctx["scope"])
	}
}

func TestLoggerConcurrentLogging(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.InfoLevel)
	logger := FromZap(zap.New(core))

	const workers = 8
	const perWorker = 25

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()

			workerLogger := logger.With(String("worker", strconv.Itoa(worker)))
			for i := 0; i < perWorker; i++ {
				workerLogger.Info("parallel log", Int("index", i))
			}
		}()
	}

	wg.Wait()

	entries := observed.All()
	want := workers * perWorker
	if len(entries) != want {
		t.Fatalf("entries count = %d, want %d", len(entries), want)
	}
}

func TestLoggerEnabled(t *testing.T) {
	t.Parallel()

	core, _ := observer.New(zapcore.InfoLevel)
	logger := FromZap(zap.New(core))

	if logger.Enabled(zap.DebugLevel) {
		t.Fatal("Enabled(debug) = true, want false")
	}

	if !logger.Enabled(zap.InfoLevel) {
		t.Fatal("Enabled(info) = false, want true")
	}
}

func TestMapProducesDeterministicFields(t *testing.T) {
	t.Parallel()

	fields := Map(map[string]any{
		"zeta":  1,
		"alpha": 2,
		"beta":  3,
	})

	if len(fields) != 3 {
		t.Fatalf("fields count = %d, want 3", len(fields))
	}

	if fields[0].Key != "alpha" || fields[1].Key != "beta" || fields[2].Key != "zeta" {
		t.Fatalf("field order = %q, %q, %q", fields[0].Key, fields[1].Key, fields[2].Key)
	}
}

func TestFieldHelpers(t *testing.T) {
	t.Parallel()

	errField := Err(assertErr{})
	if errField.Key != "error" {
		t.Fatalf("Err key = %q, want error", errField.Key)
	}

	if Action("signup").Key != ActionKey {
		t.Fatalf("Action key = %q", Action("signup").Key)
	}

	if RequestID("req-1").Key != RequestIDKey {
		t.Fatalf("RequestID key = %q", RequestID("req-1").Key)
	}
}

type assertErr struct{}

func (assertErr) Error() string {
	return "boom"
}

func ExampleLogger_Info() {
	appLogger := FromZap(zap.NewExample()).
		WithRequestID("req-42").
		WithActorID("user-7")

	appLogger.Info(
		"token issued",
		Action("token.issue"),
		Result("success"),
		String("scope", "access"),
	)

	// Output:
	// {"level":"info","msg":"token issued","request_id":"req-42","actor_id":"user-7","action":"token.issue","result":"success","scope":"access"}
}

func ExampleInterface() {
	var appLogger Interface = FromZap(zap.NewExample())

	appLogger = appLogger.WithAction("service_start").WithResult("success")
	appLogger.Info("bootstrap complete")

	// Output:
	// {"level":"info","msg":"bootstrap complete","action":"service_start","result":"success"}
}
