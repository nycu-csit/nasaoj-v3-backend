package middlewares

import (
	"net/http"

	"github.com/uptrace/bunrouter"
)

func CacheControlHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		w.Header().Set("Cache-Control", "no-store")

		return next(w, req)
	}
}
