package main

import (
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/mux"
	"github.com/overdone/stubrouter/config"
	"github.com/overdone/stubrouter/stubs"
	"github.com/overdone/stubrouter/utils"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type UserSessionData struct {
	Username string
	Jwt      string
}

type ErrorViewData struct {
	Code    int
	Message string
}

type IndexViewData struct {
	config.StubRouterConfig
	Username string
}

var cfg *config.StubRouterConfig
var stubStore stubs.StubStorage
var sessionManager *scs.SessionManager

func errorHandler(w http.ResponseWriter, r *http.Request, code int, message string) {
	data := ErrorViewData{
		Code:    code,
		Message: message,
	}
	tmpl, e := template.ParseFiles("./static/error.html")
	if e != nil {
		http.Error(w, "Server error", http.StatusBadGateway)
	}
	e = tmpl.Execute(w, data)
	if e != nil {
		http.Error(w, "Server error", http.StatusBadGateway)
	}
}

func getSessionData(r *http.Request) *UserSessionData {
	val := sessionManager.Get(r.Context(), "userData")
	data, ok := val.(*UserSessionData)

	if !ok {
		return nil
	}

	return data
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	path := utils.ParseForkPath(r.URL.Path)
	host := cfg.Targets[path]

	sessionData := getSessionData(r)

	targetPath := strings.TrimPrefix(r.URL.Path, path)
	targetUrl, _ := url.Parse(host)
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	r.URL.Scheme = targetUrl.Scheme
	r.URL.Host = targetUrl.Host
	r.URL.Path = targetPath
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Header.Set("Authorization", fmt.Sprint("Bearer ", sessionData.Jwt))
	r.Host = targetUrl.Host

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf(">>> Error while proxy requeat to [%s]: %s", targetUrl, err)
		errorHandler(w, r, http.StatusBadGateway, fmt.Sprintf("Can`t proxy request to %s", targetUrl))
	}

	if sm, err := stubStore.GetServiceMap(r.URL); err != nil || sm == nil {
		log.Println(">>> Error while getting stub")
		proxy.ServeHTTP(w, r)
	} else if stub, ok := sm.Service[targetPath]; ok {
		w.WriteHeader(stub.Code)
		for k, v := range stub.Headers {
			w.Header().Add(k, v)
		}
		w.Write([]byte(stub.Data))
	} else {
		proxy.ServeHTTP(w, r)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("./static/index.html")
	if err != nil {
		log.Panic("Server error")
	}
	sessionData := getSessionData(r)
	data := IndexViewData{*cfg, sessionData.Username}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Panic("Server error")
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	if _, hasKey := cfg.Targets[utils.ParseForkPath(r.URL.Path)]; !hasKey {
		errorHandler(w, r, http.StatusNotFound, fmt.Sprintf("Target path '%s' not found", r.URL.Path))
	} else {
		handleProxy(w, r)
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
	case "POST":
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "./static/login.html")
	case "POST":
		username := r.FormValue("username")
		if username == "" {
			log.Panic("Username must not be empty")
		}

		jwt := utils.GetTokenString(cfg, username)
		d := UserSessionData{username, jwt}
		sessionManager.Destroy(r.Context())
		sessionManager.Put(r.Context(), "userData", d)

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	se := sessionManager.Destroy(r.Context())
	if se != nil {
		log.Panic("Session error")
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func serverErrorMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				errorHandler(w, r, http.StatusInternalServerError, fmt.Sprintf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func authMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if sessionManager.Exists(r.Context(), "userData") {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		}
	}

	return http.HandlerFunc(fn)
}

func logMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func init() {
	gob.Register(&UserSessionData{})

	log.Println("-- Read config file --")
	c, err := config.ParseConfig()
	cfg = c
	if err != nil {
		log.Fatal(">>> Error while reading config file")
	}

	log.Println("-- Init stub storage --")
	switch cfg.Stubs.Storage.Type {
	case "file":
		stubStore = &stubs.FileStubStorage{FsPath: cfg.Stubs.Storage.Path}
		if cfg.Stubs.Storage.Cache.Enabled {
			stubStore = &stubs.CachedStorage{Store: stubStore}
		}
	case "redis":
		stubStore = &stubs.RedisStubStorage{ConnString: cfg.Stubs.Storage.Path}
		if cfg.Stubs.Storage.Cache.Enabled {
			stubStore = &stubs.CachedStorage{Store: stubStore}
		}
	default:
		log.Fatalf(">>> Config error. Stub storage type %s not supported", cfg.Stubs.Storage.Type)
	}
	err = stubStore.InitStorage(cfg)
	if err != nil {
		log.Fatalf(">>> Init stub store error: %s", err)
	}

	log.Println("-- Init session manager --")
	sessionManager = scs.New()
	sessionManager.Lifetime, err = time.ParseDuration(cfg.Session.Duration)
	sessionManager.IdleTimeout, err = time.ParseDuration(cfg.Session.IdleTimeout)
	sessionManager.Cookie.Name = cfg.Session.CookieName
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	sessionManager.Cookie.Secure = true

	if err != nil {
		log.Fatal(">>> Config error. Invalid config param")
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))

	mx := mux.NewRouter()
	mx.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs)).Methods("GET")
	mx.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	mx.HandleFunc("/logout", logoutHandler).Methods("GET")
	mx.Handle("/stubapi/", authMiddleware(http.HandlerFunc(apiHandler))).Methods("GET", "POST", "DELETE")
	mx.Handle("/", authMiddleware(http.HandlerFunc(rootHandler))).Methods("GET")
	mx.PathPrefix("/{route}").Handler(authMiddleware(http.HandlerFunc(routeHandler)))
	handler := serverErrorMiddleware(logMiddleware(mx))

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	log.Printf("-- Start proxy server on %s --", addr)
	if err := http.ListenAndServe(addr, sessionManager.LoadAndSave(handler)); err != nil {
		log.Printf(">>> Fail start server on %s", addr)
		log.Printf("Error: %s", err)
	}
}
