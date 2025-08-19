package middlewares

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
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

func ErrorHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		err := next(w, req)

		var httpErr HTTPError
		switch err := err.(type) {
		case nil:
			return nil

		case HTTPError:
			httpErr = err

		default:
			httpErr = NewHTTPError(err)
		}

		if httpErr.OriginError != nil {
			fmt.Fprintf(os.Stderr, "%s, %s\n", httpErr.Message, httpErr.OriginError)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", httpErr.Message)
		}
		w.WriteHeader(httpErr.StatusCode)
		_ = bunrouter.JSON(w, httpErr)

		return err
	}
}
