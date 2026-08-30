package transport

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (t *Transport) InitRouter() *mux.Router {
	router := mux.NewRouter()

	router.Handle("/api/parse", http.HandlerFunc(t.parser.Parse))
	router.Handle("/api/parse-legislation", http.HandlerFunc(t.parser.ParseLegislation))
	router.Handle("/api/parse-wiki", http.HandlerFunc(t.parser.ParseWiki))
	router.Handle("/api/parse-rules", http.HandlerFunc(t.parser.ParseRules))
	router.Handle("/api/parse-commands", http.HandlerFunc(t.parser.ParseCommands))
	router.Handle("/api/search", t.AuthMiddleware(http.HandlerFunc(t.searcher.Search)))
	router.Handle("/api/register", http.HandlerFunc(t.auth.Register))
	router.Handle("/api/login", http.HandlerFunc(t.auth.Login))
	router.Handle("/api/refresh/logout", http.HandlerFunc(t.auth.Logout))
	router.Handle("/api/refresh", http.HandlerFunc(t.auth.Refresh))
	router.Handle("/api/isadmin", t.AuthMiddleware(http.HandlerFunc(t.auth.IsAdmin)))
	router.Handle("/api/user/me", t.AuthMiddleware(http.HandlerFunc(t.auth.MeHandler)))
	return router
}
