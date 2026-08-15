package transport

import "net/http"

func (p *Parser) Parse(w http.ResponseWriter, r *http.Request) {
	if err := p.serv.Parse(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)

	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))

}
