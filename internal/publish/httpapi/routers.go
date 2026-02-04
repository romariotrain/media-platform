package httpapi

import "net/http"

// NewRouter создаёт HTTP-роутер для Publish сервиса
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// GET /publications?id=<uuid>
	mux.HandleFunc("/publications", h.GetPublication)

	// GET /publications/list?asset_id=<uuid>
	mux.HandleFunc("/publications/list", h.ListPublications)

	return mux
}
