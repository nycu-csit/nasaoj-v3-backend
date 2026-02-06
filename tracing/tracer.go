package tracing

import (
	"context"

	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	Tracer trace.Tracer
	Logger *otelzap.Logger

	version string = "v2.1.0-rc0"
)

func InitTracer(appname string) {
	Tracer = otel.Tracer(appname)

	var l *zap.Logger
	var err error
	if viper.GetBool("zap-production") {
		l, err = zap.NewProduction()
	} else {
		l, err = zap.NewDevelopment()
	}
	if err != nil {
		panic(err)
	}
	Logger = otelzap.New(l)
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
