package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net/url"
	"strings"
)

func GetTokenString(secret string, username string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func ParseForkPath(path string) string {
	return "/" + strings.Split(strings.TrimLeft(path, "/"), "/")[0]
}

func HostToString(host *url.URL) string {
	return fmt.Sprintf("%s_%s", host.Hostname(), host.Port())
}
