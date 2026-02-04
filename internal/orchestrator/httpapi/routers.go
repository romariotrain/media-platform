package httpapi

import "net/http"

// NewRouter создаёт HTTP-роутер для Orchestrator сервиса
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// POST /sagas — создать новую сагу
	// GET /sagas?id=<uuid> — получить сагу по ID
	mux.HandleFunc("/sagas", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateSaga(w, r)
		case http.MethodGet:
			h.GetSaga(w, r)
		default:
			writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// GET /sagas/list?user_id=<string>
	mux.HandleFunc("/sagas/list", h.ListSagas)

	return mux
}
