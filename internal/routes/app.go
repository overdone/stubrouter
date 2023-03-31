package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/overdone/stubrouter/internal/config"
	"github.com/overdone/stubrouter/internal/utils"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

type IndexesFileSystem struct {
	fs http.FileSystem
}

type IndexViewData struct {
	config.StubRouterConfig
	Username string
}

func (nfs IndexesFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, _ := f.Stat()
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}

func RootHandler(cfg *config.StubRouterConfig, sessionManager *scs.SessionManager) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./web/templates/index.html")
		if err != nil {
			log.Panic("Server error")
			return
		}

		username := ""
		if cfg.Auth.Enabled {
			sessionData := getSessionDataForRequest(r, sessionManager)
			username = sessionData.Username
		}

		data := IndexViewData{*cfg, username}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Panic("Server error")
		}
	}

	return fn
}

func LoginHandler(cfg *config.StubRouterConfig, sessionManager *scs.SessionManager) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			http.ServeFile(w, r, "./web/static/login.html")
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

	return fn
}

func LogoutHandler(sessionManager *scs.SessionManager) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		se := sessionManager.Destroy(r.Context())
		if se != nil {
			log.Panic("Session error")
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}

	return fn
}
