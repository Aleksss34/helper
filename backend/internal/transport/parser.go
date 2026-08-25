package transport

import (
	"log/slog"
	"net/http"
)

func (p *Parser) ParseWiki(w http.ResponseWriter, r *http.Request) {

	if err := p.serv.ParseWiki(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))

}

func (p *Parser) ParseLegislation(w http.ResponseWriter, r *http.Request) {
	if err := p.serv.ParseLegislation(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (p *Parser) ParseRules(w http.ResponseWriter, r *http.Request) {
	if err := p.serv.ParseRules(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
func (p *Parser) ParseCommands(w http.ResponseWriter, r *http.Request) {
	if err := p.serv.ParseCommands(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.log.Error("gg", slog.Any("error", err))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
func (p *Parser) Parse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := p.serv.ParseCommands(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.serv.ParseRules(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.serv.ParseWiki(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.serv.ParseLegislation(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
