package config

import (
	"testing"

	"github.com/BALLUUNN/authStartGrowingUp/pkg/logger"
)

func TestLoggerConfigFromEnvDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := loggerConfigFromEnv(mapLookup(nil))
	if err != nil {
		t.Fatalf("loggerConfigFromEnv() error = %v", err)
	}

	if cfg.ServiceName != "authStartGrowingUp" {
		t.Fatalf("ServiceName = %q", cfg.ServiceName)
	}

	if cfg.Environment != logger.Development {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
}

func TestLoggerConfigFromEnvUsesValues(t *testing.T) {
	t.Parallel()

	cfg, err := loggerConfigFromEnv(mapLookup(map[string]string{
		"APP_NAME":                "auth-api",
		"APP_ENV":                 "production",
		"LOG_LEVEL":               "warn",
		"LOG_FORMAT":              "json",
		"LOG_OUTPUT_PATHS":        "stdout, ./logs/app.log",
		"LOG_ERROR_OUTPUT_PATHS":  "stderr, ./logs/error.log",
		"LOG_DISABLE_CALLER":      "true",
		"LOG_DISABLE_STACKTRACE":  "true",
		"LOG_SAMPLING_INITIAL":    "50",
		"LOG_SAMPLING_THEREAFTER": "10",
	}))
	if err != nil {
		t.Fatalf("loggerConfigFromEnv() error = %v", err)
	}

	if cfg.ServiceName != "auth-api" {
		t.Fatalf("ServiceName = %q", cfg.ServiceName)
	}

	if cfg.Environment != logger.Production {
		t.Fatalf("Environment = %q", cfg.Environment)
	}

	if cfg.Level != "warn" || cfg.Format != logger.JSON {
		t.Fatalf("Level/Format = %q/%q", cfg.Level, cfg.Format)
	}

	if len(cfg.OutputPaths) != 2 || cfg.OutputPaths[1] != "./logs/app.log" {
		t.Fatalf("OutputPaths = %v", cfg.OutputPaths)
	}

	if len(cfg.ErrorOutputPaths) != 2 || cfg.ErrorOutputPaths[1] != "./logs/error.log" {
		t.Fatalf("ErrorOutputPaths = %v", cfg.ErrorOutputPaths)
	}

	if !cfg.DisableCaller || !cfg.DisableStacktrace {
		t.Fatalf("Disable flags = %v/%v", cfg.DisableCaller, cfg.DisableStacktrace)
	}

	if cfg.Sampling == nil || cfg.Sampling.Initial != 50 || cfg.Sampling.Thereafter != 10 {
		t.Fatalf("Sampling = %+v", cfg.Sampling)
	}
}

func TestLoggerConfigFromEnvRejectsInvalidBool(t *testing.T) {
	t.Parallel()

	_, err := loggerConfigFromEnv(mapLookup(map[string]string{
		"LOG_DISABLE_CALLER": "sometimes",
	}))
	if err == nil {
		t.Fatal("loggerConfigFromEnv() error = nil, want error")
	}
}

func TestLoggerConfigFromEnvRejectsPartialSampling(t *testing.T) {
	t.Parallel()

	_, err := loggerConfigFromEnv(mapLookup(map[string]string{
		"LOG_SAMPLING_INITIAL": "100",
	}))
	if err == nil {
		t.Fatal("loggerConfigFromEnv() error = nil, want error")
	}
}

func TestLoggerConfigFromEnvRejectsUnsupportedFormatInZapConfig(t *testing.T) {
	t.Parallel()

	cfg, err := loggerConfigFromEnv(mapLookup(map[string]string{
		"LOG_FORMAT": "pretty",
	}))
	if err != nil {
		t.Fatalf("loggerConfigFromEnv() error = %v", err)
	}

	if _, err := cfg.ZapConfig(); err == nil {
		t.Fatal("ZapConfig() error = nil, want error")
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
