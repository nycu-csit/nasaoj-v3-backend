package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
)

func CheckPreSharedKey(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		authHeader := strings.Split(req.Header.Get("Authorization"), " ")
		authAns := []string{"Bearer", viper.GetString("preshared-key")}

		if slices.Compare(authHeader, authAns) != 0 {
			fmt.Fprintf(os.Stderr, "auth failed\n")
			return HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "preshare key check failed",
			}
		}

		return next(w, req)
	}
}
