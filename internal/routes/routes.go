package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/mux"
	"github.com/overdone/stubrouter/internal/config"
	"github.com/overdone/stubrouter/internal/stubs"
	"net/http"
)

func Routes(cfg *config.StubRouterConfig, sessionManager *scs.SessionManager, stubStore stubs.StubStorage) *mux.Router {
	fs := http.FileServer(IndexesFileSystem{http.Dir("./web/static")})
	mx := mux.NewRouter()

	mx.PathPrefix("/static/").
		Handler(http.StripPrefix("/static/", fs)).
		Methods("GET")
	mx.Handle("/login", LoginHandler(cfg, sessionManager)).
		Methods("GET", "POST")
	mx.Handle("/logout", LogoutHandler(sessionManager)).
		Methods("GET")
	mx.Handle("/stubapi/", ApiHandlerTargetStub(stubStore)).
		Queries("target", "{target}", "path", "{path}").
		Methods("GET", "POST", "DELETE")
	mx.Handle("/stubapi/", ApiHandlerTarget(stubStore)).
		Queries("target", "{target}").
		Methods("GET")
	mx.Handle("/", AuthMiddleware(sessionManager)(RootHandler(cfg, sessionManager))).
		Methods("GET")
	mx.PathPrefix("/{route}").
		Handler(AuthMiddleware(sessionManager)(RouteHandler(cfg, stubStore, sessionManager)))

	mx.Use(serverErrorMiddleware, logMiddleware)

	return mx
}
