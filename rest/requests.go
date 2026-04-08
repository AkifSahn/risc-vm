package rest

type LoadProgramRequest struct {
	ProgramStr string `json:"program_str"`
}

type UpdateConfigRequest struct {
	MemorySize   uint32 `json:"memory_size"`
	PredictorBit uint8  `json:"predictor_bit"`
	Forwarding   bool   `json:"forwarding"`
}
