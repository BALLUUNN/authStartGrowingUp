// Package logger provides a structured logging wrapper around zap.
package logger

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is an alias for zap.Field for convenience.
type Field = zap.Field

// Environment defines the runtime environment for the logger.
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
	Test        Environment = "test"
)

// Format defines the log output format.
type Format string

const (
	Console Format = "console" // Human-readable format
	JSON    Format = "json"    // Machine-readable JSON format
)

// Common field keys for structured logging.
const (
	ActionKey      = "action"
	ActorIDKey     = "actor_id"
	EnvironmentKey = "environment"
	RequestIDKey   = "request_id"
	ResultKey      = "result"
	ServiceKey     = "service"
)

// SamplingConfig controls log sampling to reduce volume in high-load systems.
type SamplingConfig struct {
	Initial    int // Number of messages to log per interval before sampling
	Thereafter int // Number of messages to log after initial limit is reached
}

// Config holds all configuration options for the logger.
type Config struct {
	ServiceName       string          // Name of the service (added to all logs)
	Environment       Environment     // Runtime environment (development, production, test)
	Level             string          // Log level (debug, info, warn, error)
	Format            Format          // Output format (console, json)
	OutputPaths       []string        // Where to write logs (stdout, file paths)
	ErrorOutputPaths  []string        // Where to write internal logger errors
	DisableCaller     bool            // Disable caller info (file:line)
	DisableStacktrace bool            // Disable stacktrace capture
	Sampling          *SamplingConfig // Log sampling configuration (production only)
}

// Interface defines the package-level logging contract so consumers can swap
// the implementation without changing application code.
type Interface interface {
	Named(name string) Interface
	With(fields ...Field) Interface
	WithFields(fields ...Field) Interface
	WithRequestID(requestID string) Interface
	WithActorID(actorID string) Interface
	WithAction(action string) Interface
	WithResult(result string) Interface
	Enabled(level zapcore.Level) bool
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Sync() error
	Close() error
}

// Logger wraps zap.Logger with additional convenience methods.
type Logger struct {
	base *zap.Logger
}

var _ Interface = (*Logger)(nil)

// New creates a new Logger instance from the provided configuration.
func New(cfg Config) (*Logger, error) {
	zapCfg, err := cfg.ZapConfig()
	if err != nil {
		return nil, err
	}

	base, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return &Logger{base: base}, nil
}

// Must creates a new Logger or panics on error.
func Must(cfg Config) *Logger {
	l, err := New(cfg)
	if err != nil {
		panic(err)
	}

	return l
}

// NewNop creates a no-op logger that discards all log messages.
func NewNop() *Logger {
	return &Logger{base: zap.NewNop()}
}

// FromZap wraps an existing zap.Logger, or returns a no-op logger if nil.
func FromZap(base *zap.Logger) *Logger {
	if base == nil {
		return NewNop()
	}

	return &Logger{base: base}
}

// Zap returns the underlying zap.Logger, never nil (returns no-op if unset).
func (l *Logger) Zap() *zap.Logger {
	if l == nil || l.base == nil {
		return zap.NewNop()
	}

	return l.base
}

// Sugar returns a sugared logger for printf-style and key-value logging.
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.Zap().Sugar()
}

// Named adds a sub-logger name for hierarchical logging.
func (l *Logger) Named(name string) Interface {
	return &Logger{base: l.Zap().Named(name)}
}

// With returns a child logger with additional fields.
func (l *Logger) With(fields ...Field) Interface {
	return &Logger{base: l.Zap().With(fields...)}
}

// WithFields is an alias for With.
func (l *Logger) WithFields(fields ...Field) Interface {
	return l.With(fields...)
}

// WithRequestID adds a request ID field for request-scoped logging.
func (l *Logger) WithRequestID(requestID string) Interface {
	return l.With(RequestID(requestID))
}

// WithActorID adds an actor ID field for user-scoped logging.
func (l *Logger) WithActorID(actorID string) Interface {
	return l.With(ActorID(actorID))
}

// WithAction adds an action field for operation-scoped logging.
func (l *Logger) WithAction(action string) Interface {
	return l.With(Action(action))
}

// WithResult adds a result field for outcome-scoped logging.
func (l *Logger) WithResult(result string) Interface {
	return l.With(Result(result))
}

// Enabled reports whether the given log level is currently enabled.
func (l *Logger) Enabled(level zapcore.Level) bool {
	return l.Zap().Core().Enabled(level)
}

// Debug logs a message at Debug level with structured fields.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.Zap().Debug(msg, fields...)
}

// Info logs a message at Info level with structured fields.
func (l *Logger) Info(msg string, fields ...Field) {
	l.Zap().Info(msg, fields...)
}

// Warn logs a message at Warn level with structured fields.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.Zap().Warn(msg, fields...)
}

// Error logs a message at Error level with structured fields.
func (l *Logger) Error(msg string, fields ...Field) {
	l.Zap().Error(msg, fields...)
}

// Debugw logs a message at Debug level using key-value pairs (sugared).
func (l *Logger) Debugw(msg string, keysAndValues ...any) {
	l.Sugar().Debugw(msg, keysAndValues...)
}

// Infow logs a message at Info level using key-value pairs (sugared).
func (l *Logger) Infow(msg string, keysAndValues ...any) {
	l.Sugar().Infow(msg, keysAndValues...)
}

// Warnw logs a message at Warn level using key-value pairs (sugared).
func (l *Logger) Warnw(msg string, keysAndValues ...any) {
	l.Sugar().Warnw(msg, keysAndValues...)
}

// Errorw logs a message at Error level using key-value pairs (sugared).
func (l *Logger) Errorw(msg string, keysAndValues ...any) {
	l.Sugar().Errorw(msg, keysAndValues...)
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	err := l.Zap().Sync()
	if shouldIgnoreSyncError(err) {
		return nil
	}

	return err
}

// Close is an alias for Sync that implements io.Closer.
func (l *Logger) Close() error {
	return l.Sync()
}

// ZapConfig converts the Config into a zap.Config.
func (cfg Config) ZapConfig() (zap.Config, error) {
	normalized := cfg.withDefaults()

	if err := normalized.validate(); err != nil {
		return zap.Config{}, err
	}

	parsedLevel, err := zapcore.ParseLevel(normalized.Level)
	if err != nil {
		return zap.Config{}, fmt.Errorf("parse log level %q: %w", normalized.Level, err)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	zapCfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(parsedLevel),
		Development:       normalized.Environment == Development,
		DisableCaller:     normalized.DisableCaller,
		DisableStacktrace: normalized.DisableStacktrace,
		Sampling:          normalized.samplingConfig(),
		Encoding:          string(normalized.Format),
		EncoderConfig:     encoderConfig,
		OutputPaths:       slices.Clone(normalized.OutputPaths),
		ErrorOutputPaths:  slices.Clone(normalized.ErrorOutputPaths),
		InitialFields: map[string]any{
			ServiceKey:     normalized.ServiceName,
			EnvironmentKey: string(normalized.Environment),
		},
	}

	return zapCfg, nil
}

// withDefaults applies sensible defaults to the configuration.
func (cfg Config) withDefaults() Config {
	normalized := cfg

	normalized.ServiceName = strings.TrimSpace(normalized.ServiceName)
	if normalized.ServiceName == "" {
		normalized.ServiceName = "application"
	}

	normalized.Environment = Environment(strings.ToLower(strings.TrimSpace(string(normalized.Environment))))
	normalized.Format = Format(strings.ToLower(strings.TrimSpace(string(normalized.Format))))
	normalized.Level = strings.ToLower(strings.TrimSpace(normalized.Level))

	if normalized.Environment == "" {
		normalized.Environment = Development
	}

	if normalized.Format == "" {
		if normalized.Environment == Production {
			normalized.Format = JSON
		} else {
			normalized.Format = Console
		}
	}

	if normalized.Level == "" {
		if normalized.Environment == Production {
			normalized.Level = zap.InfoLevel.String()
		} else {
			normalized.Level = zap.DebugLevel.String()
		}
	}

	normalized.OutputPaths = normalizePaths(normalized.OutputPaths)
	if len(normalized.OutputPaths) == 0 {
		normalized.OutputPaths = []string{"stdout"}
	}

	normalized.ErrorOutputPaths = normalizePaths(normalized.ErrorOutputPaths)
	if len(normalized.ErrorOutputPaths) == 0 {
		normalized.ErrorOutputPaths = []string{"stderr"}
	}

	// Enable sampling by default in production to reduce log volume.
	if normalized.Environment == Production && normalized.Sampling == nil {
		normalized.Sampling = &SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
	}

	return normalized
}

// validate ensures the configuration values are supported.
func (cfg Config) validate() error {
	switch cfg.Environment {
	case Development, Production, Test:
	default:
		return fmt.Errorf("unsupported environment %q", cfg.Environment)
	}

	switch cfg.Format {
	case Console, JSON:
	default:
		return fmt.Errorf("unsupported log format %q", cfg.Format)
	}

	if cfg.Sampling != nil {
		if cfg.Sampling.Initial < 1 {
			return fmt.Errorf("sampling initial must be >= 1, got %d", cfg.Sampling.Initial)
		}
		if cfg.Sampling.Thereafter < 0 {
			return fmt.Errorf("sampling thereafter must be >= 0, got %d", cfg.Sampling.Thereafter)
		}
	}

	return nil
}

// samplingConfig converts the custom sampling config to zap's format.
func (cfg Config) samplingConfig() *zap.SamplingConfig {
	if cfg.Sampling == nil {
		return nil
	}

	return &zap.SamplingConfig{
		Initial:    cfg.Sampling.Initial,
		Thereafter: cfg.Sampling.Thereafter,
	}
}

// ---------- Field Constructors ----------

// Action returns a field with the action key.
func Action(value string) Field {
	return zap.String(ActionKey, value)
}

// ActorID returns a field with the actor_id key.
func ActorID(value string) Field {
	return zap.String(ActorIDKey, value)
}

// RequestID returns a field with the request_id key.
func RequestID(value string) Field {
	return zap.String(RequestIDKey, value)
}

// Result returns a field with the result key.
func Result(value string) Field {
	return zap.String(ResultKey, value)
}

// String returns a field with a custom string key and value.
func String(key, value string) Field {
	return zap.String(key, value)
}

// Bool returns a field with a custom bool key and value.
func Bool(key string, value bool) Field {
	return zap.Bool(key, value)
}

// Int returns a field with a custom int key and value.
func Int(key string, value int) Field {
	return zap.Int(key, value)
}

// Int64 returns a field with a custom int64 key and value.
func Int64(key string, value int64) Field {
	return zap.Int64(key, value)
}

// Duration returns a field with a custom duration key and value.
func Duration(key string, value time.Duration) Field {
	return zap.Duration(key, value)
}

// Any returns a field with a custom any key and value (uses reflection).
func Any(key string, value any) Field {
	return zap.Any(key, value)
}

// Err returns a field for logging errors.
func Err(err error) Field {
	return zap.Error(err)
}

// Map converts a map to a sorted slice of fields for structured logging.
func Map(values map[string]any) []Field {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	fields := make([]Field, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, Any(key, values[key]))
	}

	return fields
}

func normalizePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	return normalized
}

// shouldIgnoreSyncError checks if a sync error should be ignored (e.g., invalid fd).
func shouldIgnoreSyncError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, os.ErrInvalid) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid argument") || strings.Contains(msg, "bad file descriptor")
}
