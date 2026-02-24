package utils

import (
	"io"
	"net/http"
)

func Success(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err := io.WriteString(w,
		`{"code":200,"message":"success"}`+"\n")
	return err
}
