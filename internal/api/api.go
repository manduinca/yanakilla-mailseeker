package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/manduinca/yanakilla-mailseeker/internal/mailparse"
	"github.com/manduinca/yanakilla-mailseeker/internal/zinc"
)

const (
	defaultSize = 20
	maxSize     = 100
)

type Server struct {
	client *zinc.Client
	index  string
	assets http.Handler
}

func NewServer(client *zinc.Client, index string, assets http.Handler) *Server {
	return &Server{client: client, index: index, assets: assets}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.health)
		r.Get("/search", s.search)
	})

	if s.assets != nil {
		r.Handle("/*", s.assets)
	}

	return r
}

type searchResult struct {
	ID      string  `json:"id"`
	Score   float64 `json:"score"`
	Date    string  `json:"date"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Cc      string  `json:"cc"`
	Subject string  `json:"subject"`
	Folder  string  `json:"folder"`
	Content string  `json:"content"`
}

type searchResponse struct {
	Query string         `json:"query"`
	Took  int            `json:"took"`
	Total int            `json:"total"`
	From  int            `json:"from"`
	Size  int            `json:"size"`
	Hits  []searchResult `json:"hits"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	field := strings.TrimSpace(r.URL.Query().Get("field"))
	from := intParam(r, "from", 0, 0, 10000)
	size := intParam(r, "size", defaultSize, 1, maxSize)

	req := zinc.SearchRequest{
		SearchType: "match",
		Query:      zinc.SearchQuery{Term: q, Field: field},
		From:       from,
		MaxResults: size,
		Source:     []string{},
	}
	if q == "" {
		req.SearchType = "matchall"
	}

	res, err := s.client.Search(s.index, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "no se pudo consultar el índice")
		return
	}

	hits := make([]searchResult, 0, len(res.Hits.Hits))
	for _, h := range res.Hits.Hits {
		var e mailparse.Email
		if err := json.Unmarshal(h.Source, &e); err != nil {
			continue
		}
		hits = append(hits, searchResult{
			ID:      h.ID,
			Score:   h.Score,
			Date:    e.Date,
			From:    e.From,
			To:      e.To,
			Cc:      e.Cc,
			Subject: e.Subject,
			Folder:  e.Folder,
			Content: e.Content,
		})
	}

	writeJSON(w, http.StatusOK, searchResponse{
		Query: q,
		Took:  res.Took,
		Total: res.Hits.Total.Value,
		From:  from,
		Size:  size,
		Hits:  hits,
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "index": s.index})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func intParam(r *http.Request, name string, fallback, min, max int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
