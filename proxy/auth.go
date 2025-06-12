package proxy

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func checkAuth(username string, password string, req *http.Request) bool {
	authHeader := req.Header.Get("Proxy-Authorization")
	if authHeader == "" {
		return false
	}

	prefix := "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}

	encoded := strings.TrimPrefix(authHeader, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == username && parts[1] == password
}
