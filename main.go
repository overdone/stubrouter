package main

import (
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
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

func renderError(w http.ResponseWriter, code int, message string) {
	data := ErrorViewData{
		Code:    code,
		Message: message,
	}
	tmpl, e := template.ParseFiles("./templates/error.html")
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

func getSessionData(r *http.Request) *UserSessionData {
	val := sessionManager.Get(r.Context(), "userData")
	data, ok := val.(*UserSessionData)

	if !ok {
		return nil
	}

	return data
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	path := "/" + mux.Vars(r)["route"]
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
		renderError(w, http.StatusBadGateway, fmt.Sprintf("Can`t proxy request to %s", targetUrl))
	}

	if sm, err := stubStore.GetServiceStubs(r.URL); err != nil || sm == nil {
		proxy.ServeHTTP(w, r)
	} else if stub, ok := sm.Service[targetPath]; ok {
		log.Printf("Get %s response from stub", targetPath)
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
	tmpl, err := template.ParseFiles("./templates/index.html")
	if err != nil {
		log.Panic("Server error")
		return
	}
	sessionData := getSessionData(r)
	data := IndexViewData{*cfg, sessionData.Username}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Panic("Server error")
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	forkPath := "/" + mux.Vars(r)["route"]

	if _, hasKey := cfg.Targets[forkPath]; !hasKey {
		msg := fmt.Sprintf("Target path '%s' not found", r.URL.Path)
		log.Println(msg)
		renderError(w, http.StatusNotFound, msg)
		return
	}

	if forkPath == r.URL.Path {
		// Go to index
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
	} else {
		handleProxy(w, r)
	}
}

func apiHandlerTarget(w http.ResponseWriter, r *http.Request) {
	targetParam := mux.Vars(r)["target"]
	targetUrl, err := url.Parse(targetParam)

	notFoundMessage := fmt.Sprintf("Stubs for target %s not found", targetUrl)

	if err != nil {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		if sm, err := stubStore.GetServiceStubs(targetUrl); err == nil && sm != nil {
			w.Header().Set("Content-Type", "application/json")
			resp, err := json.Marshal(sm.Service)
			if err != nil {
				http.Error(w, fmt.Sprintf("Can`t parse stubs for target %s", targetUrl), http.StatusInternalServerError)
			}
			w.Write(resp)
		} else {
			w.Write([]byte("{}"))
		}
	}
}

func apiHandlerTargetStub(w http.ResponseWriter, r *http.Request) {
	targetParam := mux.Vars(r)["target"]
	pathParam := mux.Vars(r)["path"]
	targetUrl, err := url.Parse(targetParam)

	notFoundMessage := fmt.Sprintf("Stub %s for target %s not found", pathParam, targetUrl)

	if err != nil {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		resp := []byte("")
		if sm, err := stubStore.GetServiceStubs(targetUrl); err == nil && sm != nil {
			w.Header().Set("Content-Type", "application/json")
			stub, ok := sm.Service[pathParam]
			if ok {
				if resp, err = json.Marshal(stub); err != nil {
					http.Error(w, fmt.Sprintf("Can`t parse stubs for target %s", targetUrl), http.StatusInternalServerError)
				}
				w.Write(resp)
			} else {
				http.Error(w, notFoundMessage, http.StatusNotFound)
			}
		}

	case "POST":
		var reqData map[string]interface{}

		if err = json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			http.Error(w, "Request data not valid", http.StatusBadRequest)
			return
		}

		code, err := strconv.Atoi(reqData["code"].(string))
		data := reqData["data"].(string)
		if err != nil {
			http.Error(w, "Request data not valid", http.StatusBadRequest)
			return
		}

		var headers map[string]string
		if err = json.Unmarshal([]byte(reqData["headers"].(string)), &headers); err != nil {
			http.Error(w, "Request data not valid", http.StatusBadRequest)
			return
		}

		stubData := stubs.ServiceStub{Code: code, Data: data, Headers: headers}
		err = stubStore.SaveServiceStub(targetUrl, pathParam, stubData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			http.Error(w, "", http.StatusOK)
		}

	case "DELETE":
		err = stubStore.RemoveServiceStub(targetUrl, pathParam)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			http.Error(w, "", http.StatusOK)
		}
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

		val := sessionManager.Pop(r.Context(), "originUrl")
		tUrl, ok := val.(string)
		if !ok {
			tUrl = "/"
		}

		jwt := utils.GetTokenString(cfg, username)
		d := UserSessionData{username, jwt}
		sessionManager.Destroy(r.Context())
		sessionManager.Put(r.Context(), "userData", d)

		http.Redirect(w, r, tUrl, http.StatusMovedPermanently)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	se := sessionManager.Destroy(r.Context())
	if se != nil {
		log.Panic("Session error")
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
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

func authMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if sessionManager.Exists(r.Context(), "userData") {
			next.ServeHTTP(w, r)
		} else {
			// Save original user url path
			sessionManager.Put(r.Context(), "originUrl", r.URL.Path)
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

	if err != nil {
		log.Fatal(">>> Config error. Invalid config param")
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	mx := mux.NewRouter()

	mx.PathPrefix("/static/").
		Handler(http.StripPrefix("/static/", fs)).
		Methods("GET")
	mx.HandleFunc("/login", loginHandler).
		Methods("GET", "POST")
	mx.HandleFunc("/logout", logoutHandler).
		Methods("GET")
	mx.Handle("/stubapi/", http.HandlerFunc(apiHandlerTargetStub)).
		Queries("target", "{target}", "path", "{path}").
		Methods("GET", "POST", "DELETE")
	mx.Handle("/stubapi/", http.HandlerFunc(apiHandlerTarget)).
		Queries("target", "{target}").
		Methods("GET")
	mx.Handle("/", authMiddleware(http.HandlerFunc(rootHandler))).
		Methods("GET")
	mx.PathPrefix("/{route}").
		Handler(authMiddleware(http.HandlerFunc(routeHandler)))

	handler := serverErrorMiddleware(logMiddleware(mx))

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	log.Printf("-- Start proxy server on %s --", addr)
	if err := http.ListenAndServe(addr, sessionManager.LoadAndSave(handler)); err != nil {
		log.Printf(">>> Fail start server on %s", addr)
		log.Printf("Error: %s", err)
	}
}
