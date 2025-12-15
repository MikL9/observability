package utils

import (
	"net/http"
	"strings"
)

func GetRealIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		ips := strings.Split(ip, ",")
		return ips[0]
	}
	return ""
}
