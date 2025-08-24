package middlewares

import (
	"net/http"
	"sync"

	"github.com/uptrace/bunrouter"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type HTTPError struct {
	StatusCode  int    `json:"code"`
	Message     string `json:"message"`
	OriginError error  `json:"-"`
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

var (
	once   sync.Once
	logger *otelzap.Logger
)

func ErrorHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		err := next(w, req)
		ctx := req.Context()
		span := trace.SpanFromContext(ctx)
		once.Do(func() {
			l, err := zap.NewDevelopment()
			if err != nil {
				panic(err)
			}
			logger = otelzap.New(l)
		})

		var httpErr HTTPError
		switch err := err.(type) {
		case nil:
			return nil

		case HTTPError:
			httpErr = err
			span.SetStatus(codes.Error, err.Error())

		default:
			httpErr = NewHTTPError(err)
		}

		if httpErr.OriginError != nil {
			logger.Ctx(ctx).Error(httpErr.Message,
				zap.Error(httpErr.OriginError),
			)
		} else {
			logger.Ctx(ctx).Error(httpErr.Message)
		}
		w.WriteHeader(httpErr.StatusCode)
		_ = bunrouter.JSON(w, httpErr)

		return err
	}
}
