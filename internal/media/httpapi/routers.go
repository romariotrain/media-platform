package httpapi

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// POST /media
	mux.HandleFunc("/media", h.CreateMedia)

	// GET /media/{id}
	// Важно: trailing slash, чтобы handler мог TrimPrefix("/media/")
	mux.HandleFunc("/media/", h.GetMedia)

	return mux
}
