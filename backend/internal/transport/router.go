package transport

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (t *Transport) InitRouter() *mux.Router {
	router := mux.NewRouter()

	router.Handle("/parse", http.HandlerFunc(t.parser.Parse))
	router.Handle("/parse-legislation", http.HandlerFunc(t.parser.ParseLegislation))
	router.Handle("/parse-wiki", http.HandlerFunc(t.parser.ParseWiki))
	router.Handle("/search", http.HandlerFunc(t.searcher.Search))
	return router
}
