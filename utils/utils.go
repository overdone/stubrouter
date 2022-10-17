package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/overdone/stubrouter/config"
	"net/url"
	"strings"
)

func GetTokenString(cfg *config.StubRouterConfig, username string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		cfg.Session.UseridField: username,
	})
	tokenString, _ := token.SignedString([]byte(cfg.Session.TokenSecret))
	return tokenString
}

func ParseForkPath(path string) string {
	return "/" + strings.Split(strings.TrimLeft(path, "/"), "/")[0]
}

func HostToString(host *url.URL) string {
	return fmt.Sprintf("%s_%s", host.Hostname(), host.Port())
}
