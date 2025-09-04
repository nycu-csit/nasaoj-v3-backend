package middlewares

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

func CheckPrivs(
	reqPrivs models.Privileges,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			// fmt.Fprintf(os.Stderr, "[INFO] Context: %v\n", req.Context())
			privs, ok := req.Context().
				Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "privs missing",
				}
			}

			// General Privileges
			for i, field := range reflect.VisibleFields(
				reflect.TypeOf(models.Privileges{})) {

				matched, err := regexp.Match(`Can*`, []byte(field.Name))
				if err != nil || !matched || field.Type.Name() != "int" {
					continue
				}

				if reflect.ValueOf(reqPrivs).Field(i).Interface().(int) >
					reflect.ValueOf(privs).Field(i).Interface().(int) {
					return HTTPError{
						StatusCode: http.StatusForbidden,
						Message:    fmt.Sprintf("privs '%s' not met", field.Name),
					}
				}
			}

			return next(w, req)
		}
	}
}
