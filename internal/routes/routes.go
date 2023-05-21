package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/overdone/stubrouter/internal/config"
	"github.com/overdone/stubrouter/internal/stubs"
	goji "goji.io"
	"goji.io/pat"
)

func Routes(cfg *config.StubRouterConfig, sessionManager *scs.SessionManager, stubStore stubs.StubStorage) *goji.Mux {
	router := goji.NewMux()

	router.HandleFunc(pat.New("/static/*"), StaticHandler())

	router.Handle(pat.Get("/"), authMiddleware(cfg, sessionManager)(RootHandler(cfg, sessionManager)))

	loginFunc := LoginHandler(cfg, sessionManager)
	router.Handle(pat.Get("/login"), loginFunc)
	router.Handle(pat.Post("/login"), loginFunc)

	router.Handle(pat.Get("/logout"), LogoutHandler(sessionManager))

	router.Handle(pat.New("/stubapi/*"), StubApiHandler(stubStore))

	routHandler := authMiddleware(cfg, sessionManager)(RouteHandler(cfg, stubStore, sessionManager))
	router.Handle(pat.New("/:route"), routHandler)
	router.Handle(pat.New("/:route/*"), routHandler)

	router.Use(serverErrorMiddleware)
	router.Use(logMiddleware)

	return router
}
