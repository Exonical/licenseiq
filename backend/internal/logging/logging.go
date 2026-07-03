package logging

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string, dev bool) (*zap.Logger, error) {
	parsedLevel := zap.NewAtomicLevelAt(zap.InfoLevel)
	if err := parsedLevel.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level)))); err != nil {
		parsedLevel.SetLevel(zap.InfoLevel)
	}

	if dev {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = parsedLevel
		cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
		return cfg.Build(zap.AddCaller())
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = parsedLevel
	return cfg.Build(zap.AddCaller())
}
