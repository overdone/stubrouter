package routes

import (
	"crypto/tls"
	"fmt"
	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/mux"
	"github.com/overdone/stubrouter/internal/config"
	"github.com/overdone/stubrouter/internal/stubs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func handleProxy(cfg *config.StubRouterConfig, stubStore stubs.StubStorage, sessionManager *scs.SessionManager) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		path := "/" + mux.Vars(r)["route"]
		host := cfg.Targets[path]

		sessionData := getSessionDataForRequest(r, sessionManager)

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
			log.Panic(fmt.Sprintf("Can`t proxy request to %s", targetUrl))
		}

		if sm, err := stubStore.GetServiceStubs(r.URL); err != nil || sm == nil {
			proxy.ServeHTTP(w, r)
		} else if stub, ok := sm.Service[targetPath]; ok {
			log.Printf("Get %s response from stub", targetPath)
			w.WriteHeader(stub.Code)
			for k, v := range stub.Headers {
				w.Header().Add(k, v)
			}
			time.Sleep(time.Duration(stub.Timeout) * time.Millisecond)
			w.Write([]byte(stub.Data))
		} else {
			proxy.ServeHTTP(w, r)
		}
	}

	return fn
}

func RouteHandler(cfg *config.StubRouterConfig, stubStore stubs.StubStorage, sessionManager *scs.SessionManager) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		forkPath := "/" + mux.Vars(r)["route"]

		if _, hasKey := cfg.Targets[forkPath]; !hasKey {
			msg := fmt.Sprintf("Target path '%s' not found", r.URL.Path)
			log.Panic(msg)
			return
		}

		if forkPath == r.URL.Path {
			// Go to index
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		} else {
			handleProxy(cfg, stubStore, sessionManager)(w, r)
		}
	}

	return fn
}
