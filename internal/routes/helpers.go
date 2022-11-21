package routes

import (
	"github.com/alexedwards/scs/v2"
	"net/http"
)

type UserSessionData struct {
	Username string
	Jwt      string
}

func getSessionDataForRequest(r *http.Request, sessionManager *scs.SessionManager) *UserSessionData {
	val := sessionManager.Get(r.Context(), "userData")
	data, ok := val.(*UserSessionData)

	if !ok {
		return nil
	}

	return data
}
