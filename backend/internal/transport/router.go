package transport

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (t *Transport) InitRouter() *mux.Router {
	router := mux.NewRouter()

	router.Handle("/parse", http.HandlerFunc(t.parser.Parse))
	router.Handle("/search", http.HandlerFunc(t.searcher.Search))
	return router
}
