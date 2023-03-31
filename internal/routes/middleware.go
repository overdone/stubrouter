package routes

import (
	"fmt"
	"github.com/alexedwards/scs/v2"
	"github.com/overdone/stubrouter/internal/config"
	"html/template"
	"log"
	"net/http"
)

type ErrorViewData struct {
	Code    int
	Message string
}

func renderError(w http.ResponseWriter, code int, message string) {
	data := ErrorViewData{
		Code:    code,
		Message: message,
	}
	tmpl, e := template.ParseFiles("./web/templates/error.html")
	if e != nil {
		http.Error(w, "Server error", http.StatusBadGateway)
		return
	}
	e = tmpl.Execute(w, data)
	if e != nil {
		http.Error(w, "Server error", http.StatusBadGateway)
		return
	}
}

func serverErrorMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				renderError(w, http.StatusInternalServerError, fmt.Sprintf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func AuthMiddleware(cfg *config.StubRouterConfig, sessionManager *scs.SessionManager) func(http.Handler) http.Handler {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Auth.Enabled || sessionManager.Exists(r.Context(), "userData") {
				next.ServeHTTP(w, r)
			} else {
				// Save original user url path
				sessionManager.Put(r.Context(), "originUrl", r.URL.Path)
				http.Redirect(w, r, "/login", http.StatusMovedPermanently)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}

func logMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
