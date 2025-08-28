package utils

import (
	"fmt"
	"net/http"
)

func Redirect(w http.ResponseWriter, loc string) error {
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusFound)
	_, err := fmt.Fprintf(w, "302 Found")
	return err
}
