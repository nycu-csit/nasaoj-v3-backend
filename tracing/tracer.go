package tracing

import (
	"context"

	"github.com/spf13/viper"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	Tracer trace.Tracer

	appname string = "nasaoj"
	version string = "v2.0.0"
)

func Init() {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(viper.GetString("uptrace-dsn")),
		uptrace.WithServiceName(appname),
		uptrace.WithServiceVersion(version),
	)

	Tracer = otel.Tracer(appname)
}

func Cleanup(ctx context.Context) {
	uptrace.Shutdown(ctx)
}
