package httpapi

import "net/http"

// NewRouter создаёт HTTP-роутер для Ingest сервиса
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// GET /upload-token?token=abc123
	mux.HandleFunc("/upload-token", h.GetUploadToken)

	// POST /upload?token=abc123
	mux.HandleFunc("/upload", h.Upload)

	return mux
}
