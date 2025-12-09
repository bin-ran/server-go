package utils

import (
	"net/http"
	"strings"
)

func ParseIP(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}
