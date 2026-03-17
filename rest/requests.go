package rest

type LoadProgramRequest struct {
	ProgramStr string `json:"program_str"`
}

type UpdateConfigRequest struct {
	Forwarding       bool `json:"forwarding"`
	BranchPrediction bool `json:"branch_prediction"`
}
