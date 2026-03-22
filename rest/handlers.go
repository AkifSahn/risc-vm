package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AkifSahn/risc-vm/vm"
)

func ListenAndServe(addr string) {
	mux := SetupRoutes()

	http.ListenAndServe(addr, corsMiddleware(mux))
}

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/session/{id}", getSessionHandler)
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

	data := NewSessionResponse{Id: id}
	writeJSON(w, http.StatusOK, GenericResponse{"OK", data})
}

// GET /api/session
//
// Checks if session exists
func getSessionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	mu.Lock()
	defer mu.Unlock()

	_, ok := sessions[id]
	if !ok {
		http.Error(w, fmt.Sprintf("Session not found! 'id=%s'", id), http.StatusNotFound)
		return
	}

	// TODO: we should return the state of the state and maybe the program?? Maybe?
	writeJSON(w, http.StatusNoContent, nil)
}

// POST /api/session/{id}/load_program
func loadProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	var req LoadProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	session.Reset()
	err := session.LoadProgramFromStr(req.ProgramStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse given program: %s", err.Error()), http.StatusBadRequest)
		return
	}

	prog_str := session.GetProgramStr()
	data := struct {
		Program []string `json:"program"`
	}{
		Program: prog_str[1:],
	}

	writeJSON(w, http.StatusOK, GenericResponse{"OK", data})
}

func updateConfigHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
}

// POST /api/session/{id}/step
//
// Forwards the program by one cycle and returns any updated state.
// For now, we return the whole state but later we will only return the diff
func stepProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	session.ExecuteCycle()

	data := session.GetState()
	writeJSON(w, http.StatusOK, GenericResponse{"OK", data})
}

// --------- Instruction Handlers ---------

func getInstructionList(w http.ResponseWriter, r *http.Request) {
	insts := vm.GetInstructionStringList()

	data := ListInstructionsResponse{Instructions: insts}
	writeJSON(w, http.StatusOK, GenericResponse{"OK", data})
}
