package httpapi

import "net/http"

// cors wraps handler with CORS headers
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewRouter создаёт HTTP-роутер для Publish сервиса
func NewRouter(h *Handler, publishDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// GET /publications?id=<uuid>
	mux.HandleFunc("/publications", h.GetPublication)

	// GET /publications/list?asset_id=<uuid>
	mux.HandleFunc("/publications/list", h.ListPublications)

	// GET /static/<asset_id>/<file> — отдача опубликованных файлов
	fileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(publishDir)))
	mux.Handle("/static/", fileServer)

	return cors(mux)
}
