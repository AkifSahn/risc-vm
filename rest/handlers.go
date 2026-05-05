package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, "Invalid request body!"})
		return
	}

	config, err := vm.CreateConfig(req.MemorySize, req.MemorySize, req.PredictorBit, req.Forwarding, req.PredictorBit > 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, err.Error()})
	}

	id, err := newSession(*config)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, err.Error()})
		return
	}

	data := NewSessionResponse{Id: id}
	writeJSON(w, http.StatusOK, GenericResponse{"OK", data, ""})
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
		s := fmt.Sprintf("Session not found! 'id=%s'", id)
		writeJSON(w, http.StatusNotFound, GenericResponse{"", nil, s})
		return
	}

	// TODO: we should return the state of the state and maybe the program?? Maybe?
	writeJSON(w, http.StatusNoContent, nil)
}

// POST /api/session/{id}/load_program
func loadProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	var req LoadProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, "Invalid request body!"})
		return
	}

	// Reset the vm and reload the program given
	session.Reset(session.Config)
	err := session.LoadProgramFromStr(req.ProgramStr)
	if err != nil {
		s := fmt.Sprintf("Failed to parse: %v", err.Error())
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, s})
		return
	}

	prog_str := session.GetProgramStr()
	data := struct {
		Program []string `json:"program"`
	}{
		Program: prog_str[1:],
	}

	writeJSON(w, http.StatusOK, GenericResponse{"OK", data, ""})
}

func updateConfigHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, "Invalid request body!"})
		return
	}

	config, err := vm.CreateConfig(req.MemorySize, req.MemorySize, req.PredictorBit, req.Forwarding, req.PredictorBit > 0)
	if err != nil {
		s := fmt.Sprintf("Failed to update config: %v", err.Error())
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, s})
		return
	}

	session.Reset(*config)

	writeJSON(w, http.StatusOK, GenericResponse{"OK", nil, ""})
}

// POST /api/session/{id}/step?n
//
// Forwards the program by one cycle and returns any updated state.
// For now, we return the whole state but later we will only return the diff
func stepProgramHandler(w http.ResponseWriter, r *http.Request, session *vm.Vm) {
	params := r.URL.Query()
	n_str := params.Get("n")
	n, err := strconv.Atoi(n_str)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, GenericResponse{"", nil, "Url parameter 'n' is not a number!"})
		return
	}

	// Negative 'n' number means execute until halt.
	// TODO: what if the program contains an infinite loop??
	states := []vm.Vm_State{}
	for i := n; i != 0 && !session.Halted; i -= 1 {
		session.ExecuteCycle()
		state := session.GetState()
		states = append(states, state)
	}

	if session.Runtime_error != nil {
		writeJSON(w, http.StatusOK, GenericResponse{"", states, session.Runtime_error.Error()})
		return
	}

	writeJSON(w, http.StatusOK, GenericResponse{"OK", states, ""})
}

// --------- Instruction Handlers ---------

func getInstructionList(w http.ResponseWriter, r *http.Request) {
	insts := vm.GetInstructionStringList()

	data := ListInstructionsResponse{Instructions: insts}
	writeJSON(w, http.StatusOK, GenericResponse{"OK", data, ""})
}
