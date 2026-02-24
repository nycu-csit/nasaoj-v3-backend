package utils

import (
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
)

func Error404(w http.ResponseWriter, req bunrouter.Request) error {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "application/json")
	_, err := io.WriteString(w,
		`{"code":404,"message":"Not Found"}`+"\n")
	return err
}
