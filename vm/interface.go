package vm

type Vm_State struct {
	Pc        uint32          `json:"pc"`
	Cycle     int             `json:"cycle"`
	Fetched   int             `json:"inst_fetched"`
	Retired   int             `json:"inst_retired"`
	Stalled   int             `json:"inst_stalled"`
	Cpi       float32         `json:"cpi"`
	Registers map[uint8]int32 `json:"registers"`
	Memory    map[uint32]byte `json:"memory"`
	CycleInfo Cycle_Info      `json:"cycle_info"`
	Halt      bool            `json:"halt"`
}

func (v *Vm) GetState() Vm_State {

	state := Vm_State{
		Pc:        v.Pc,
		Cycle:     int(v.Dm.N_cycle),
		Fetched:   int(v.Dm.N_fetched),
		Retired:   int(v.Dm.N_retired),
		Stalled:   int(v.Dm.N_stalls),
		Cpi:       v.Dm.CalculateCpi(),
		Registers: map[uint8]int32{},
		Memory:    map[uint32]byte{},
		CycleInfo: v.Dm.Cycle_infos[len(v.Dm.Cycle_infos)-1],
		Halt:      v.Halted,
	}

	// TODO: Fix, load byte by byte not word
	for _, addr := range v.Memory_diff_addr {
		state.Memory[addr] = v.Memory[addr]
	}

	for _, reg := range v.Register_diff_idx {
		state.Registers[reg] = v.Registers[reg].Data
	}

	return state
}

func (v *Vm) GetProgramStr() []string {
	var result []string
	for _, inst := range v.program {
		result = append(result, inst.Str())
	}

	return result
}
