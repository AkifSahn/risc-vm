package rest

import (
	"encoding/json"
	"net/http"

	"github.com/AkifSahn/risc-vm/vm"
)

type VmStateResponse struct {
	Pc        int32             `json:"pc"`
	Cycle     int               `json:"cycle"` // Which cycle we are currently at
	Registers map[uint8]int32   `json:"registers"`
	Memory    map[uint32][]byte `json:"memory"`

	Stages [5]uint32 `json:"stages"`
}

type NewSessionResponse struct {
	Id string `json:"session_id"`
}

type ProgramResponse struct {
	Program []vm.Instruction `json:"program"`
}

type ListInstructionsResponse struct {
	Instructions []string `json:"instructions"`
}

type GenericResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewGenericResponse(msg string, data any) GenericResponse {
	return GenericResponse{
		Message: msg,
		Data:    data,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
