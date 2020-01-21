package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewStandardLogger creates a new zap.Logger based on common configuration
//
// This is intended to be used with zap.ReplaceGlobals() in an application's
// main.go.
func NewStandardLogger(logLevel zapcore.Level) (l *zap.Logger, err error) {
	config := NewStandardZapConfig(logLevel)
	return config.Build()
}

// NewStandardZapConfig returns a sensible [config](https://godoc.org/go.uber.org/zap#Config) for a Zap logger.
func NewStandardZapConfig(logLevel zapcore.Level) zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}
