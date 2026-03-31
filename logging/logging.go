package logging

import (
	"log/slog"

	"github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Debug bool
	Level string
}

func NewLogger(cfg Config) (*zap.Logger, error) {
	zapConfig := newLoggerConfigDev()
	if !cfg.Debug {
		zapConfig = newLoggerConfigProduction()
	}

	var err error
	zapConfig.Level, err = zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	return zapConfig.Build()
}

func NewSugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}

func NewStdLogger(logger *zap.Logger) *slog.Logger {
	return slog.New(slogzap.Option{Level: slog.LevelDebug, Logger: logger}.NewZapHandler())
}

func newLoggerConfigDev() *zap.Config {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapConfig.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	return &zapConfig
}

func newLoggerConfigProduction() *zap.Config {
	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	return &zapConfig
}
