package middlewares

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
)

func AccessLog(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		tracing.Logger.Ctx(req.Context()).
			Info(req.Method + " " + req.Params().Route())
		return next(w, req)
	}
}
