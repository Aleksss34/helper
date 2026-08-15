package transport

import (
	"encoding/json"
	"fmt"
	"gateway/backend/internal/dto"
	"log/slog"
	"net/http"
)

type Resp struct {
	Response string `json:"response"`
}

func (s *Searcher) Search(w http.ResponseWriter, r *http.Request) {
	var req dto.ReqSearch
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&req)
	outChan := make(chan string)
	go func() {
		defer close(outChan)
		if err := s.serv.Search(r.Context(), req.Question, outChan); err != nil {
			s.log.Error("Search failed", slog.Any("error", err))
		}
	}()

	for chunk := range outChan {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
	}

}
