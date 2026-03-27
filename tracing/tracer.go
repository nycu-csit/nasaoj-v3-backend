package tracing

import (
	"context"

	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Tracer  trace.Tracer
	Logger  *otelzap.Logger
	appName string

	version string = "v3.0.0"
)

// InitTracer initializes the tracer and a basic logger without OTel log export.
// Call InitLogger after InitUptrace to enable log export to Uptrace.
func InitTracer(name string) {
	appName = name
	Tracer = otel.Tracer(appName)
	Logger = otelzap.New(newZapLogger().Named(appName))
}

// InitLogger re-initializes the logger with the OpenTelemetry log provider.
// This must be called after InitUptrace to properly export logs.
func InitLogger() {
	Logger = otelzap.New(newZapLogger().Named(appName),
		otelzap.WithLoggerProvider(global.GetLoggerProvider()),
	)
}

func newZapLogger() *zap.Logger {
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
	return l
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

func getLogSeverity() log.Severity {
	switch viper.GetString("log-level") {
	case "debug":
		return log.SeverityDebug
	case "info":
		return log.SeverityInfo
	case "warn":
		return log.SeverityWarn
	case "error":
		return log.SeverityError
	default:
		return log.SeverityInfo
	}
}

func InitUptrace(appname, system string) {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(viper.GetString("uptrace-dsn")),
		uptrace.WithServiceName(appname),
		uptrace.WithServiceVersion(version),
		uptrace.WithLoggingEnabled(true),
		uptrace.WithMinLogSeverity(getLogSeverity()),
		uptrace.WithResourceAttributes(
			attribute.String("system", system),
		),
	)
}

func DeferUptrace(ctx context.Context) {
	uptrace.Shutdown(ctx)
}
