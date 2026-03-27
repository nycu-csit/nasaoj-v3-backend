package tracing

import (
	"context"
	"fmt"

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
		otelzap.WithMinLevel(getLogLevel()),
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

// SpanEvent adds an event to the current span from context.
// Use this when you want the log to appear directly in the span in Uptrace.
// It also logs to the standard logger for local output.
func SpanEvent(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(msg, trace.WithAttributes(attrs...))
	}
	// Also log to standard logger for local output
	Logger.Ctx(ctx).Debug(msg)
}

// SpanEventWithFields adds an event to the current span with zap-style fields.
// Converts zap fields to OTel attributes for span events.
func SpanEventWithFields(ctx context.Context, msg string, fields ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := zapFieldsToAttributes(fields)
		span.AddEvent(msg, trace.WithAttributes(attrs...))
	}
	// Also log to standard logger for local output
	Logger.Ctx(ctx).Debug(msg, fields...)
}

// zapFieldsToAttributes converts zap fields to OTel attributes.
func zapFieldsToAttributes(fields []zap.Field) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(fields))
	for _, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			attrs = append(attrs, attribute.String(f.Key, f.String))
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			attrs = append(attrs, attribute.Int64(f.Key, f.Integer))
		case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			attrs = append(attrs, attribute.Int64(f.Key, int64(f.Integer)))
		case zapcore.Float64Type:
			attrs = append(attrs, attribute.Float64(f.Key, float64(f.Integer)))
		case zapcore.BoolType:
			attrs = append(attrs, attribute.Bool(f.Key, f.Integer == 1))
		case zapcore.ErrorType:
			if f.Interface != nil {
				attrs = append(attrs, attribute.String(f.Key, f.Interface.(error).Error()))
			}
		default:
			// For complex types, use string representation
			if f.Interface != nil {
				attrs = append(attrs, attribute.String(f.Key, fmt.Sprintf("%v", f.Interface)))
			}
		}
	}
	return attrs
}
