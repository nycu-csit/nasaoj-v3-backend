package utils

import (
	"net"

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

	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	return host
}
