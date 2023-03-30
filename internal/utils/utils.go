package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/overdone/stubrouter/internal/config"
	"net/url"
)

func GetTokenString(cfg *config.StubRouterConfig, username string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		cfg.Session.UseridField: username,
	})
	tokenString, _ := token.SignedString([]byte(""))
	return tokenString
}

func HostToString(host *url.URL) string {
	return fmt.Sprintf("%s_%s_%s", host.Scheme, host.Hostname(), host.Port())
}
