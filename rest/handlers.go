package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AkifSahn/risc-vm/vm"
)

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/session/new", newSessionHandler)
	mux.HandleFunc("POST /api/session/{id}/load_program", withSessionMiddleware(loadProgramHandler))
	mux.HandleFunc("POST /api/session/{id}/update_config", withSessionMiddleware(updateConfigHandler))
	mux.HandleFunc("POST /api/session/{id}/step", withSessionMiddleware(stepProgramHandler))

	mux.HandleFunc("GET /api/instructions", getInstructionList)

	return mux
}

// --------- Session Handlers ---------

// POST /api/session/new
//
// Creates a session and returns the session id.
func newSessionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := newSession()
	if err != nil {
		http.Error(w, "Failed to create session.", http.StatusInternalServerError)
		return
	}

	response := NewSessionResponse{Id: id}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /api/session/{id}/load_program
func loadProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	var req LoadProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := session.LoadProgramFromStr(req.ProgramStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse given program: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateConfigHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
}

// POST /api/session/{id}/step
//
// Forwards the program by one cycle and returns any updated state.
// For now, we return the whole state but later we will only return the diff
func stepProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	session.ExecuteCycle()

	response := session.GetState()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --------- Instruction Handlers ---------

func getInstructionList(w http.ResponseWriter, r *http.Request) {
	insts := vm.GetInstructionStringList()

	response := ListInstructionsResponse{
		Instructions: insts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
