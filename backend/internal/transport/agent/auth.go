package agent

import (
	"crypto/subtle"
	"strings"
)

type PlatformAuthenticator struct {
	token string
}

func NewPlatformAuthenticator(token string) PlatformAuthenticator {
	return PlatformAuthenticator{token: token}
}

func (a PlatformAuthenticator) ValidHeader(authHeader string) bool {
	token := platformBearerToken(authHeader)
	if a.token == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) == 1
}

func platformBearerToken(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return auth
}
