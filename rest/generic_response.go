package rest

import (
	"encoding/json"
	"net/http"
)

type GenericResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
	Error   string `json:"error"`
}

func NewGenericResponse(msg string, data any, err string) GenericResponse {
	return GenericResponse{
		Message: msg,
		Data:    data,
		Error:   err,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
