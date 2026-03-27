package middlewares

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type HTTPError struct {
	StatusCode  int    `json:"code"`
	Message     string `json:"message"`
	OriginError error  `json:"-"`

	TraceID string `json:"traceId"`
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(err error) HTTPError {
	return HTTPError{
		StatusCode:  http.StatusInternalServerError,
		Message:     "Internal server error with unknown reason",
		OriginError: err,
	}
}

func ErrorHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		err := next(w, req)
		ctx := req.Context()

		var httpErr HTTPError
		switch err := err.(type) {
		case nil:
			return nil

		case HTTPError:
			httpErr = err

		default:
			tracing.Logger.Ctx(ctx).
				Error("unhandled error",
					zap.Error(err))
			httpErr = NewHTTPError(err)
		}

		// https://opentelemetry.io/docs/specs/semconv/http/http-spans/#status
		span := trace.SpanFromContext(ctx)
		httpErr.TraceID = span.SpanContext().TraceID().String()
		span.SetAttributes(
			attribute.Int("http.response.status_code", httpErr.StatusCode),
		)

		if 500 <= httpErr.StatusCode && httpErr.StatusCode <= 599 {
			span.SetStatus(codes.Error, err.Error())
			if httpErr.OriginError == nil {
				tracing.Logger.Ctx(ctx).
					Error(httpErr.Message)
				span.RecordError(httpErr)
			} else {
				tracing.Logger.Ctx(ctx).
					Error(httpErr.Message,
						zap.Error(httpErr.OriginError))
				span.RecordError(httpErr.OriginError)
			}
		} else {
			span.SetStatus(codes.Ok, "")
			if httpErr.OriginError == nil {
				tracing.Logger.Ctx(ctx).
					Warn(httpErr.Message)
			} else {
				tracing.Logger.Ctx(ctx).
					Warn(httpErr.Message,
						zap.Error(httpErr.OriginError))
			}
		}

		w.WriteHeader(httpErr.StatusCode)
		_ = bunrouter.JSON(w, httpErr)

		return err
	}
}
