package middlewares

import (
	"net/http"

	"github.com/uptrace/bunrouter"
)

func CORSHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			return next(w, req)
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if req.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST")
			w.Header().Set("Access-Control-Allow-Headers",
				"authorization,content-type,content-length")
			return nil
		}

		return next(w, req)
	}
}
