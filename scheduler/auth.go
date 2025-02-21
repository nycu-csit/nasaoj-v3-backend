package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/uptrace/bunrouter"
)

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func Auth(req bunrouter.Request) error {
	PRESHARED_KEY := os.Getenv("PRESHARED_KEY")
	var auth []string = strings.Split(req.Header.Get("Authorization"), " ")

	if len(auth) < 2 || auth[0] != "Bearer" || auth[1] != PRESHARED_KEY {
		fmt.Printf("auth failed\n")
		return &errorString{"Authorization error"}
	}
	return nil
}
