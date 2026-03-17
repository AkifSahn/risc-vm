package rest

import "github.com/AkifSahn/risc-vm/vm"

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
