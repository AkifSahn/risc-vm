package vm

import (
	"fmt"
	"log"
	"strings"
)

const (
	STALL_RAW uint8 = 1 << iota
	STALL_BRANCH
)

const WORD_SIZE = 4 // In bytes

type Register struct {
	Data int32
	// This is not bool because there can be multiple instructions with same destination register
	// And one register may free it when it is actually not free.
	// So, instead of true/false, we count how many instructions still want to write to this
	// If 0, then it is free.
	Busy int32
}

type Pipeline_Buffer struct {
	pc    uint32
	inst  Instruction
	valid bool
}

type Vm_Config struct {
	Mem_size           uint32 // In bytes
	Stack_size         uint32 // In bytes
	Bp_nbit            uint8  // Branch predictor bit size
	Forwarding_enabled bool
	Bp_enabled         bool
}

func CreateConfig(mem_size, stack_size uint32, bp_nbit uint8, forwarding, branch_prediction bool) (*Vm_Config, error) {
	if mem_size%WORD_SIZE != 0 {
		return nil, fmt.Errorf("Invalid memory size '%d', Memory size must be a multiple of word size(%d).\n",
			mem_size, WORD_SIZE)
	}

	if stack_size%WORD_SIZE != 0 {
		return nil, fmt.Errorf("Invalid stack size '%d', Stack size must be a multiple of word size(%d).\n",
			mem_size, WORD_SIZE)
	}

	if stack_size > mem_size {
		return nil, fmt.Errorf("Stack(%d) size can not be bigger than memory size(%d).\n",
			stack_size, mem_size)
	}

	return &Vm_Config{
		Mem_size:           mem_size,
		Stack_size:         stack_size,
		Bp_nbit:            bp_nbit,
		Forwarding_enabled: forwarding,
		Bp_enabled:         branch_prediction,
	}, nil
}

type Vm struct {
	Config     Vm_Config
	Dm         Diagnostics_Manager
	Bp         Branch_Predictor
	cycle_info Cycle_Info

	Pc      uint32
	program []Instruction

	Registers [32]Register
	Memory    []byte

	// Memory and register diff arrays holding the updated addr/idx for memory cells and registers for the last cycle.
	// This is useful when we only want to know which memory cells and registers are changed in a cycle.
	Memory_diff_addr  []uint32
	Register_diff_idx []uint8

	_fd_buff [2]Pipeline_Buffer
	_dx_buff [2]Pipeline_Buffer
	_xm_buff [2]Pipeline_Buffer
	_mw_buff [2]Pipeline_Buffer

	// Holds the instruction execute cycle numbers.
	_instCycleTable map[Inst_Op]int

	// Bitflag for different stalls.
	// Each stall is same in principle, but their cause may be different.
	_stall_map byte

	_halt bool
}

func CreateVm(config Vm_Config) (*Vm, error) {
	vm := Vm{
		program: make([]Instruction, 0),
		Memory:  make([]byte, config.Mem_size),
		Dm:      CreateDiagnosticsManager(),
		Bp:      create_predictor(config.Bp_nbit),
		Config:  config,
	}

	// TODO: Is this good?
	vm.Dm.Forwarding_enabled = config.Forwarding_enabled
	vm.Dm.Bp_enabled = config.Bp_enabled

	// Initialize stack pointer to the MAX_ADDR
	vm.Registers[abiToRegNum["sp"]].Data = int32(config.Mem_size)

	// Fill the instCycleTable to default values
	{
		vm._instCycleTable = map[Inst_Op]int{
			Inst_Mul: 3,
			Inst_Div: 3,
			Inst_Rem: 3,
		}
	}

	return &vm, nil
}

func (v *Vm) Reset() {
	// I don't know if this is a good way to reset the vm.
	// We'll see...

	v.Dm.Reset()
	v.Bp.Reset()
	v.cycle_info = Cycle_Info{}

	v.Dm.Forwarding_enabled = v.Config.Forwarding_enabled
	v.Dm.Bp_enabled = v.Config.Bp_enabled

	v.Pc = 0
	v.program = v.program[:0]
	v.Registers = [32]Register{}

	// Clear the memory and registers
	v.Memory_diff_addr = v.Memory_diff_addr[:0]
	v.Register_diff_idx = v.Register_diff_idx[:0]

	// Reset the sp
	v.Registers[abiToRegNum["sp"]].Data = int32(v.Config.Mem_size)

	v._stall_map = 0
	v._halt = false

	// Shifting twice totally empties the both read and write buffers
	v.shiftPipelineBuffers()
	v.shiftPipelineBuffers()
}

// Returns an error if a parsing error occurs
func (v *Vm) LoadProgramFromFile(fileName string) error {
	// Parse the file etc...
	program, pc, err := ParseProgramFromFile(fileName)
	v.SetProgram(program, uint32(pc))

	return err
}

func (v *Vm) LoadProgramFromStr(programStr string) error {
	r := strings.NewReader(programStr)
	program, pc, err := ParseProgramFromReader(r)
	v.SetProgram(program, uint32(pc))

	return err
}

func (v *Vm) SetProgram(program []Instruction, pc uint32) {
	v.Pc = pc
	v.program = program

	// Reset the Diagnostics_Manager
	v.Dm.Program_size = uint(len(program))
	v.Dm.N_cycle = 0
	v.Dm.N_stalls = 0
	v.Dm.N_executed_inst = 0
}

// This function checks if a register at decode stage can be forwarded later on.
func (v *Vm) checkRegisterForwardDecode(reg int32) bool {
	if reg <= 0 {
		return true
	}

	// If forwarding is not enabled, well we can't forward
	if !v.Config.Forwarding_enabled {
		return false
	}

	half_inst := v._dx_buff[1].inst
	full_inst := v._xm_buff[1].inst // Enabled if and only if can't forward from inst_s1

	// Check if bypass source instructions actually write to a register
	if half_inst._fmt == Fmt_R || half_inst._fmt == Fmt_I || half_inst._fmt == Fmt_U || half_inst._fmt == Fmt_J {
		// Compare the destination register with given source register
		if half_inst.Op != Inst_Load && half_inst.Rd == reg {
			return true
		}
	}

	if full_inst._fmt == Fmt_R || full_inst._fmt == Fmt_I || full_inst._fmt == Fmt_U || full_inst._fmt == Fmt_J {
		// Compare the destination register with
		if full_inst.Op != Inst_Load && full_inst.Rd == reg {
			return true
		}
	}

	return false
}

// Gets the given register value from bypass.
// If register number don't match, returns (-1, false).
func (v *Vm) getRegValueFromBypass(reg int32) (int32, bypass_type, bool) {
	if reg <= 0 {
		return -1, 0, false
	}

	if !v.Config.Forwarding_enabled {
		return -1, 0, false
	}

	half_inst := v._xm_buff[1].inst
	full_inst := v._mw_buff[1].inst // Enabled only if we can't forward from inst_s1

	// Check if bypass source instructions actually write to a register
	if half_inst._fmt == Fmt_R || half_inst._fmt == Fmt_I || half_inst._fmt == Fmt_U || half_inst._fmt == Fmt_J {
		// Compare the destination register with given source register
		if half_inst.Op != Inst_Load && half_inst.Rd == reg {
			return half_inst._result, BYPASS_XM, true
		}
	}

	if full_inst._fmt == Fmt_R || full_inst._fmt == Fmt_I || full_inst._fmt == Fmt_U || full_inst._fmt == Fmt_J {
		// Compare the destination register with
		if full_inst.Op != Inst_Load && full_inst.Rd == reg {
			return full_inst._result, BYPASS_MW, true
		}
	}

	return -1, 0, false
}

// This function checks if we should stall in the decode stage or not based on the instruction.
//
// Checks for RAW(Read After Write) Data Hazard.
func (v *Vm) shouldStallDecode(inst Instruction) bool {
	// If write buff of ID/EX is valid(written by some other inst)
	if v._dx_buff[0].valid {
		return true
	}

	rs1, rs2 := inst.getSourceRegisters()
	if rs1 > 0 {
		if v.Registers[rs1].Busy > 0 {
			// FIX: Checking the inst.Op == Inst_Store is not a good approach
			// We need this check because the first source is not an ALU input and can't be forwarded
			if inst.Op == Inst_Store || !v.checkRegisterForwardDecode(rs1) {
				return true
			}
		}
	}

	if rs2 > 0 {
		if v.Registers[rs2].Busy > 0 {
			if !v.checkRegisterForwardDecode(rs2) {
				return true
			}
		}
	}

	return false
}

// flushes the IF/ID and ID/EX pipeline buffers
func (v *Vm) flush() {
	v._fd_buff[0].valid = false
	v._fd_buff[1].valid = false
}

func (v *Vm) run_fetch() {
	inst := v.program[v.Pc]

	{
		n, ok := v._instCycleTable[inst.Op]
		if !ok {
			n = 1 // Default number of cycles in ex is 1
		}
		inst._ex_remaining = n
	}

	// Update the cyle info
	{
		v.cycle_info.Stage_pcs[0] = v.Pc
	}

	v.Pc++

	v._fd_buff[0].inst = inst
	v._fd_buff[0].pc = v.Pc
	v._fd_buff[0].valid = true
}

func (v *Vm) run_decode() {
	inst := v._fd_buff[1].inst
	pc := v._fd_buff[1].pc

	if v.shouldStallDecode(inst) {
		v._stall_map |= STALL_RAW
		// Feed the IF/ID buffer to itself to avoid draining it
		v._fd_buff[0] = v._fd_buff[1]
		v._fd_buff[0].valid = true
		return
	} else if v._stall_map&STALL_RAW != 0 {
		v._stall_map &= ^STALL_RAW
	}

	// If this is a branch instruction, stall the fetch until the branch addr is calculated
	if inst.isControlInstruction() {
		// If branch prediction is enabled, do prediction.
		// If not, just stall
		if v.Config.Bp_enabled {
			taken, target := v.Bp.predict(uint32(pc - 1))

			if taken {
				v.Pc = target
			}
		} else {
			v._stall_map |= STALL_BRANCH
		}
	}

	// Update the cyle info
	v.cycle_info.Stage_pcs[1] = uint32(pc) - 1

	switch inst._fmt {
	case Fmt_R:
		inst._s1 = v.Registers[inst.Rs1].Data
		inst._s2 = v.Registers[inst.Rs2].Data

		// Set the destination register as busy
		v.Registers[inst.Rd].Busy++
		v.Register_diff_idx = append(v.Register_diff_idx, uint8(inst.Rd))
	case Fmt_I:
		// In load, immediate is placed in a different position so we check it explicitly.
		if inst.Op == Inst_Load {
			inst._imm = inst.Rs1
			inst._s1 = v.Registers[inst.Rs2].Data
		} else {
			inst._s1 = v.Registers[inst.Rs1].Data
			inst._imm = inst.Rs2
		}

		// Set the destination register as busy
		v.Registers[inst.Rd].Busy++
		v.Register_diff_idx = append(v.Register_diff_idx, uint8(inst.Rd))
	case Fmt_S: // sw s1 imm(s2) = mem[rf(s2) + imm] <- s1
		inst._s1 = v.Registers[inst.Rd].Data
		inst._imm = inst.Rs1
		inst._s2 = v.Registers[inst.Rs2].Data
	case Fmt_B:
		inst._s1 = v.Registers[inst.Rd].Data
		inst._s2 = v.Registers[inst.Rs1].Data
		inst._imm = inst.Rs2
	case Fmt_U:
		inst._imm = inst.Rs1
		v.Registers[inst.Rd].Busy++
		v.Register_diff_idx = append(v.Register_diff_idx, uint8(inst.Rd))
	case Fmt_J:
		inst._imm = inst.Rs1
		v.Registers[inst.Rd].Busy++
		v.Register_diff_idx = append(v.Register_diff_idx, uint8(inst.Rd))
	}

	v._dx_buff[0].inst = inst
	v._dx_buff[0].pc = pc
	v._dx_buff[0].valid = true
}

func (v *Vm) run_execute() {
	inst := v._dx_buff[1].inst
	pc := v._dx_buff[1].pc

	// Update the cyle info
	v.cycle_info.Stage_pcs[2] = uint32(pc) - 1

	inst._ex_remaining--

	// Handle multi cycle instructions
	// Fed the read buffer back to write if need more cycles to execute
	if inst._ex_remaining > 0 {
		v._dx_buff[0] = v._dx_buff[1]
		v._dx_buff[0].inst._ex_remaining = inst._ex_remaining // Propagate the ex_remaining
		v._dx_buff[0].valid = true
		return
	}

	var s1, s2 int32

	// Get the source values either from bypass or use the one loaded from RF
	{
		var ok bool
		var t bypass_type

		rs1, rs2 := inst.getAluInputRegisters()

		s1, t, ok = v.getRegValueFromBypass(rs1)
		if !ok {
			s1 = inst._s1
		}

		// Update cycle info
		v.cycle_info.S1_bypass_status = t

		s2, t, ok = v.getRegValueFromBypass(rs2)
		if !ok {
			s2 = inst._s2
		}

		// Update cycle info
		v.cycle_info.S2_bypass_status = t
	}

	// Only used if a branch instruction
	var branch_target uint32
	var branch_taken bool

	switch inst.Op {
	case Inst_Add:
		inst._result = s1 + s2
	case Inst_Sub:
		inst._result = s1 - s2
	case Inst_Mul:
		inst._result = s1 * s2
	case Inst_Div:
		inst._result = s1 / s2
	case Inst_Rem:
		inst._result = s1 % s2
	case Inst_Xor:
		inst._result = s1 ^ s2
	case Inst_Or:
		inst._result = s1 | s2
	case Inst_And:
		inst._result = s1 & s2

	case Inst_Addi:
		inst._result = s1 + inst._imm
	case Inst_Subi:
		inst._result = s1 - inst._imm
	case Inst_Xori:
		inst._result = s1 ^ inst._imm
	case Inst_Ori:
		inst._result = s1 | inst._imm
	case Inst_Andi:
		inst._result = s1 & inst._imm

	case Inst_Load: // load word
		addr := s1 + inst._imm
		if addr%(WORD_SIZE) != 0 {
			log.Printf("Illegal read attempt from unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
			v._halt = true
		}
		inst._result = addr

	case Inst_Jalr:
		inst._result = int32(pc)
		branch_taken = true
		branch_target = uint32(s1 + inst._imm)
		// v.pc = s1 + inst._imm

	case Inst_Slli: // rd = rs1 << imm[0:4]
		inst._result = s1 << inst._imm

	case Inst_Store: // Store word
		addr := s2 + inst._imm // In bytes

		// Calculated addr is in bytes and WORD_SIZE is in bytes. So convert WORD_SIZE to bits
		if addr%(WORD_SIZE) != 0 {
			log.Printf("ERROR - Illegal write attempt to unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
			v._halt = true
		}

		inst._result = addr // Each memory cell holds one byte

	case Inst_Beq:
		if s1 == s2 {
			branch_taken = true
		}
		branch_target = uint32(int32(pc)+inst._imm) - 1 // v.pc holds the next instruction hence -1
	case Inst_Bne:
		if s1 != s2 {
			branch_taken = true
		}
		branch_target = uint32(int32(pc) + inst._imm - 1)
	case Inst_Blt:
		if s1 < s2 {
			branch_taken = true
		}
		branch_target = uint32(int32(pc) + inst._imm - 1)
	case Inst_Bge:
		if s1 >= s2 {
			branch_taken = true
			branch_target = uint32(int32(pc) + inst._imm - 1)
		}

	case Inst_Jal: // Jump And Link
		inst._result = int32(pc)
		branch_taken = true
		branch_target = uint32(int32(pc) + inst._imm - 1)

	case Inst_Lui:
		inst._result = inst._imm
	case Inst_Auipc:
		inst._result = int32(pc) + inst._imm - 1
	}

	// Branch address is calculated, we can stop stalling.
	if inst.isControlInstruction() {
		v.Dm.N_branch++

		correct := false
		// If prediction is enabled, update the predictor and flush if necessary
		if v.Config.Bp_enabled {
			correct = v.Bp.update(uint32(pc-1), branch_target, branch_taken)
			if !correct {
				v.Dm.N_mispred++
				v.flush()
			}
		} else {
			v._stall_map &= ^STALL_BRANCH
		}

		// If not, correct do the correct branch
		// If BP is not enabled then the 'correct' will stay as false and pc will be updated.
		if !correct {
			if branch_taken {
				v.Pc = branch_target
			} else {
				v.Pc = pc
			}
		}

	}

	v._xm_buff[0].inst = inst
	v._xm_buff[0].pc = pc
	v._xm_buff[0].valid = true
}

func (v *Vm) run_memory() {
	inst := v._xm_buff[1].inst
	pc := v._xm_buff[1].pc

	// Update the cycle info
	v.cycle_info.Stage_pcs[3] = uint32(pc) - 1

	// Memory layout is little-endian
	// b3 b2 b1 b0
	switch inst.Op {
	case Inst_Store: // Store word
		u := uint32(inst._s1)
		addr := inst._result
		v.Memory[addr+0] = byte(u >> 0)
		v.Memory[addr+1] = byte(u >> 8)
		v.Memory[addr+2] = byte(u >> 16)
		v.Memory[addr+3] = byte(u >> 24)

		v.Memory_diff_addr = append(v.Memory_diff_addr, uint32(addr))
	case Inst_Load: // Load word
		addr := inst._result
		u := uint32(v.Memory[addr+0]) |
			uint32(v.Memory[addr+1])<<8 |
			uint32(v.Memory[addr+2])<<16 |
			uint32(v.Memory[addr+3])<<24

		inst._result = int32(u)
	}

	v._mw_buff[0].inst = inst
	v._mw_buff[0].pc = pc
	v._mw_buff[0].valid = true
}

func (v *Vm) run_writeback() {
	inst := v._mw_buff[1].inst
	pc := v._mw_buff[1].pc

	// Update the cycle info
	v.cycle_info.Stage_pcs[4] = uint32(pc) - 1

	if inst.Op == Inst_End {
		v._halt = true
		return
	}

	// We don't want to writeback if instruction type is S
	if inst._fmt == Fmt_R || inst._fmt == Fmt_I || inst._fmt == Fmt_U || inst._fmt == Fmt_J {
		// set the destination register as free
		v.Registers[inst.Rd].Busy--

		// We don't allow writes to x0 register
		if inst.Rd == 0 {
			return
		}

		v.Registers[inst.Rd].Data = inst._result
		v.Register_diff_idx = append(v.Register_diff_idx, uint8(inst.Rd))
	}

}

func (v *Vm) RunPipelined() {
	for !v._halt {
		v.ExecuteCycle()
	}
}

// In pipelined fashion, we may execute multiple pipeline stage
// For example, in a single cycle the processor could run: EX, D and F
// each belonging to different instructions.
//
// This functions does exactly that, each stage has it's own instruction.
// At the end of each cycle, instructions moves to the next stage in the pipeline.
func (v *Vm) ExecuteCycle() {
	v.Dm.N_cycle++

	v.cycle_info = Cycle_Info{}

	// Clear the memory and register diff
	v.Memory_diff_addr = v.Memory_diff_addr[:0]
	v.Register_diff_idx = v.Register_diff_idx[:0]

	if v._mw_buff[1].valid && !v._halt {
		v.run_writeback()
	}

	if v._xm_buff[1].valid && !v._halt {
		v.run_memory()
	}

	if v._dx_buff[1].valid && !v._halt {
		v.run_execute()
	}

	if v._fd_buff[1].valid && !v._halt {
		v.run_decode()
	}

	if v.Pc < uint32(v.Dm.Program_size) && !v._halt && v._stall_map == 0 {
		v.run_fetch()

		v.Dm.N_executed_inst++
	}

	// Save the cycle state
	{
		// For diagnostic purposes
		if v._stall_map > 0 {
			v.Dm.N_stalls++
			v.cycle_info.Stalled = true
		}

		v.Dm.Cycle_infos = append(v.Dm.Cycle_infos, v.cycle_info)

		if v.cycle_info.S1_bypass_status != 0 {
			v.Dm.N_forwards++
		}

		if v.cycle_info.S2_bypass_status != 0 {
			v.Dm.N_forwards++
		}
	}

	v.shiftPipelineBuffers()
}

/*
This function shifts the double-buffered pipeline buffers.
That means moving the WRITE buffer to its corresponding READ buffer.

After this operation, WRITE buffer becomes empty.
If WRITE buffer was already empty beforehand, this also empties the READ buffer.
Thus draining the pipeline buffer completely.

This function does not deal with stall conditions.
At stall, the READ part must fed back to the WRITE part so that the buffer is not drained.
*/
func (v *Vm) shiftPipelineBuffers() {
	v._fd_buff[1] = v._fd_buff[0]
	v._fd_buff[0].valid = false

	v._dx_buff[1] = v._dx_buff[0]
	v._dx_buff[0].valid = false

	v._xm_buff[1] = v._xm_buff[0]
	v._xm_buff[0].valid = false

	v._mw_buff[1] = v._mw_buff[0]
	v._mw_buff[0].valid = false
}
