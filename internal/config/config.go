// Package config provides configuration loading for the application.
// It reads environment variables and .env files to build structured configs.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/BALLUUNN/authStartGrowingUp/pkg/logger"
)

// LoggerConfig builds a logger.Config from environment variables.
// It loads .env file if present (ignores missing file error) and maps
// environment variables to logger configuration fields.
// Returns an error if any variable fails to parse.
func LoggerConfig() (logger.Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return logger.Config{}, fmt.Errorf("load .env: %w", err)
	}

	return loggerConfigFromEnv(os.LookupEnv)
}

// loggerConfigFromEnv populates logger.Config using a lookup function.
func loggerConfigFromEnv(lookup func(string) (string, bool)) (logger.Config, error) {
	cfg := logger.Config{
		ServiceName: envOrDefault(lookup, "APP_NAME", "authStartGrowingUp"),
		Environment: logger.Environment(envOrDefault(lookup, "APP_ENV", string(logger.Development))),
		Level:       envOrDefault(lookup, "LOG_LEVEL", ""),
		Format:      logger.Format(envOrDefault(lookup, "LOG_FORMAT", "")),
	}

	if outputPaths := splitCSV(getEnv(lookup, "LOG_OUTPUT_PATHS")); len(outputPaths) > 0 {
		cfg.OutputPaths = outputPaths
	}

	if errorPaths := splitCSV(getEnv(lookup, "LOG_ERROR_OUTPUT_PATHS")); len(errorPaths) > 0 {
		cfg.ErrorOutputPaths = errorPaths
	}

	disableCaller, err := parseOptionalBool(lookup, "LOG_DISABLE_CALLER")
	if err != nil {
		return logger.Config{}, err
	}
	cfg.DisableCaller = disableCaller

	disableStacktrace, err := parseOptionalBool(lookup, "LOG_DISABLE_STACKTRACE")
	if err != nil {
		return logger.Config{}, err
	}
	cfg.DisableStacktrace = disableStacktrace

	sampling, err := parseSampling(lookup)
	if err != nil {
		return logger.Config{}, err
	}
	cfg.Sampling = sampling

	return cfg, nil
}

// parseOptionalBool parses a boolean environment variable, returning false if unset or empty.
func parseOptionalBool(lookup func(string) (string, bool), key string) (bool, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

// parseSampling parses sampling config from two environment variables.
// Returns nil if neither is set; requires both if one is present.
func parseSampling(lookup func(string) (string, bool)) (*logger.SamplingConfig, error) {
	initialRaw, initialSet := lookup("LOG_SAMPLING_INITIAL")
	thereafterRaw, thereafterSet := lookup("LOG_SAMPLING_THEREAFTER")

	if !initialSet && !thereafterSet {
		return nil, nil
	}

	if !initialSet || !thereafterSet {
		return nil, errors.New("both LOG_SAMPLING_INITIAL and LOG_SAMPLING_THEREAFTER must be set together")
	}

	initial, err := strconv.Atoi(strings.TrimSpace(initialRaw))
	if err != nil {
		return nil, fmt.Errorf("parse LOG_SAMPLING_INITIAL: %w", err)
	}

	thereafter, err := strconv.Atoi(strings.TrimSpace(thereafterRaw))
	if err != nil {
		return nil, fmt.Errorf("parse LOG_SAMPLING_THEREAFTER: %w", err)
	}

	return &logger.SamplingConfig{
		Initial:    initial,
		Thereafter: thereafter,
	}, nil
}

// splitCSV splits a comma-separated string into trimmed non-empty parts.
func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}

// envOrDefault returns the environment variable value or fallback if empty.
func envOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	value := getEnv(lookup, key)
	if value == "" {
		return fallback
	}

	return value
}

// getEnv returns the trimmed value of an environment variable, empty if not set.
func getEnv(lookup func(string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}
