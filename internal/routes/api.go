package routes

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/overdone/stubrouter/internal/stubs"
	"net/http"
	"net/url"
	"strconv"
)

func ApiHandlerTarget(stubStore stubs.StubStorage) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
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

	return fn
}

func ApiHandlerTargetStub(stubStore stubs.StubStorage) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
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
			timeout, err := strconv.Atoi(reqData["timeout"].(string))
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

			stubData := stubs.ServiceStub{Code: code, Data: data, Headers: headers, Timeout: timeout}
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

	return fn
}
