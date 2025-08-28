package utils

import (
	"fmt"
	"net/http"
)

func Success(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprintf(w, "success")
	return err
}
