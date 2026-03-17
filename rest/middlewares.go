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
