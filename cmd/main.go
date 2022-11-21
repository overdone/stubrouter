package main

import (
	"encoding/gob"
	"fmt"
	"github.com/alexedwards/scs/v2"
	"github.com/overdone/stubrouter/internal/config"
	"github.com/overdone/stubrouter/internal/routes"
	"github.com/overdone/stubrouter/internal/stubs"
	"log"
	"net/http"
	"strconv"
	"time"
)

var cfg *config.StubRouterConfig
var stubStore stubs.StubStorage
var sessionManager *scs.SessionManager

func init() {
	gob.Register(&routes.UserSessionData{})

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
	handler := routes.Routes(cfg, sessionManager, stubStore)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	log.Printf("-- Start proxy server on %s --", addr)
	if err := http.ListenAndServe(addr, sessionManager.LoadAndSave(handler)); err != nil {
		log.Printf(">>> Fail start server on %s", addr)
		log.Printf("Error: %s", err)
	}
}
