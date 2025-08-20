package utils

import (
	"strings"

	"github.com/uptrace/bunrouter"
)

func GetRemoteAddr(req bunrouter.Request) string {
	xForwardedFor := req.Header.Get("X-Forwarded-For")
	xRealIP := req.Header.Get("X-Real-IP")

	if xForwardedFor != "" {
		return xForwardedFor
	}

	if xRealIP != "" {
		return xRealIP
	}

	return strings.Split(req.RemoteAddr, ":")[0]
}
