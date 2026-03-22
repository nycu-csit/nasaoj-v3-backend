package tracing

import (
	"context"

	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Tracer trace.Tracer
	Logger *otelzap.Logger

	version string = "v3.0.0"
)

func InitTracer(appname string) {
	Tracer = otel.Tracer(appname)

	var l *zap.Logger
	var err error
	if viper.GetBool("zap-production") {
		cfg := zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(getLogLevel())
		l, err = cfg.Build()
	} else {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(getLogLevel())
		l, err = cfg.Build()
	}
	if err != nil {
		panic(err)
	}
	Logger = otelzap.New(l)
}

func getLogLevel() zapcore.Level {
	switch viper.GetString("log-level") {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func InitUptrace(appname string) {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(viper.GetString("uptrace-dsn")),
		uptrace.WithServiceName(appname),
		uptrace.WithServiceVersion(version),
	)
}

func DeferUptrace(ctx context.Context) {
	uptrace.Shutdown(ctx)
}
