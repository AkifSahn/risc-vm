package rest

import (
	"fmt"
	"net/http"

	"github.com/AkifSahn/risc-vm/vm"
)

func withSessionMiddleware(next func(w http.ResponseWriter, r *http.Request, s *vm.Vm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		mu.Lock()
		defer mu.Unlock()

		session, ok := sessions[id]
		if !ok {
			http.Error(w, fmt.Sprintf("Session not found! 'id=%s'", id), http.StatusNotFound)
			return
		}

		next(w, r, session)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
