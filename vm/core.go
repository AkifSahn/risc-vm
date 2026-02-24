package vm

import (
	"fmt"
	"log"
)

const (
	STALL_RAW uint8 = 1 << iota
	STALL_BRANCH
)

type Instruction struct {
	Op  Inst_Op
	Rd  int32
	Rs1 int32
	Rs2 int32

	_s1           int32
	_s2           int32
	_imm          int32
	_result       int32
	_fmt          Inst_Fmt
	_ex_remaining int // Number of executions remaining
}

func newInstruction(Op Inst_Op, Rd int32, Rs1 int32, Rs2 int32) Instruction {
	return Instruction{
		Op:  Op,
		Rd:  Rd,
		Rs1: Rs1,
		Rs2: Rs2,
	}
}

// @Redundant: mostly same as getAluInputRegisters, find a way to remove one of the one of the functions
func getSourceRegisters(inst Instruction) (int32, int32) {
	switch inst._fmt {
	case Fmt_R:
		return inst.Rs1, inst.Rs2

	case Fmt_I:
		if inst.Op == Inst_Load {
			return inst.Rs2, -1
		} else {
			return inst.Rs1, -1
		}

	case Fmt_S:
		return inst.Rd, inst.Rs2

	case Fmt_B:
		return inst.Rd, inst.Rs1

	case Fmt_U, Fmt_J:
		return -1, -1

	default:
		log.Fatalf("Unexpected vm.Inst_Fmt: %#v", inst._fmt)
		return -1, -1
	}
}

func getAluInputRegisters(inst Instruction) (int32, int32) {
	switch inst._fmt {
	case Fmt_R:
		return inst.Rs1, inst.Rs2

	case Fmt_I:
		if inst.Op == Inst_Load {
			return inst.Rs2, -1
		} else {
			return inst.Rs1, -1
		}

	case Fmt_S:
		return -1, inst.Rs2

	case Fmt_B:
		return inst.Rd, inst.Rs1

	case Fmt_U, Fmt_J:
		return -1, -1

	default:
		log.Fatalf("Unexpected vm.Inst_Fmt: %#v", inst._fmt)
		return -1, -1
	}
}

// This function checks if a register at decode stage can be forwarded later on.
func checkRegisterForwardDecode(vm *Vm, reg int32) bool {
	if reg <= 0 {
		return true
	}

	half_inst := vm._dx_buff[1].inst
	full_inst := vm._xm_buff[1].inst // Enabled if and only if can't forward from inst_s1

	// Check if bypass source instructions actually write to a register
	if half_inst._fmt == Fmt_R || half_inst._fmt == Fmt_I || half_inst._fmt == Fmt_U || half_inst._fmt == Fmt_J {
		// Compare the destination register with given source register
		if half_inst.Op != Inst_Load && half_inst.Rd == reg {
			return true
		}
	}

	if full_inst._fmt == Fmt_R || full_inst._fmt == Fmt_I || full_inst._fmt == Fmt_U || full_inst._fmt == Fmt_J {
		// Compare the destination register with
		if half_inst.Op != Inst_Load && full_inst.Rd == reg {
			return true
		}
	}

	return false
}

// Gets the given register value from bypass.
// If register number don't match, returns (-1, false).
func getRegValueFromBypass(vm *Vm, reg int32) (int32, bypass_type, bool) {
	if reg <= 0 {
		return -1, 0, false
	}

	half_inst := vm._xm_buff[1].inst
	full_inst := vm._mw_buff[1].inst // Enabled only if we can't forward from inst_s1

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

type Register struct {
	Data int32
	Busy bool
}

type Pipeline_Buffer struct {
	pc    int32
	inst  Instruction
	valid bool
}

const WORD_SIZE = 4 // In bytes

type Vm struct {
	pc      int32
	program []Instruction

	registers   [32]Register
	memory      []byte
	_mem_size   int // In bytes
	_stack_size int // In bytes

	_fd_buff [2]Pipeline_Buffer
	_dx_buff [2]Pipeline_Buffer
	_xm_buff [2]Pipeline_Buffer
	_mw_buff [2]Pipeline_Buffer

	_halt bool

	// Holds the instruction execute cycle numbers.
	_instCycleTable map[Inst_Op]int

	// Bitflag for different stalls.
	// Each stall is same in principle, but their cause may be different.
	_stall_map byte

	Dm         Diagnostics_Manager
	cycle_info Cycle_Info
}

func CreateVm(mem_size, stack_size int) (*Vm, error) {
	if mem_size <= 0 || stack_size <= 0 {
		return nil, fmt.Errorf("Memory/stack size must be non-zero!")
	}

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

	vm := Vm{
		program: make([]Instruction, 0),
		memory:  make([]byte, mem_size),
		Dm:      CreateDiagnosticsManager(),

		_mem_size:   mem_size,
		_stack_size: stack_size,
	}

	// Initialize stack pointer to the MAX_ADDR
	vm.registers[abiToRegNum["sp"]].Data = int32(mem_size)

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

// Returns an error if a parsing error occurs
func (v *Vm) LoadProgramFromFile(fileName string) error {
	// Parse the file etc...
	program, pc, err := ParseProgramFromFile(fileName)
	v.SetProgram(program, pc)

	return err
}

func (v *Vm) SetProgram(program []Instruction, pc int) {
	v.pc = int32(pc)
	v.program = program

	// Reset the Diagnostics_Manager
	v.Dm.program_size = uint(len(program))
	v.Dm.n_cycle = 0
	v.Dm.n_stalls = 0
	v.Dm.n_executed_inst = 0
}

func (v *Vm) isControlInstruction(inst Instruction) bool {
	if inst._fmt == Fmt_B || inst._fmt == Fmt_J || inst.Op == Inst_Jalr {
		return true
	}

	return false
}

// This function checks if we should stall in the decode stage or not based on the instruction.
//
// Checks for RAW(Read After Write) Data Hazard.
func (v *Vm) shouldStallDecode(inst Instruction) bool {
	// If write buff of ID/EX is valid(written by some other inst)
	if v._dx_buff[0].valid {
		return true
	}

	rs1, rs2 := getSourceRegisters(inst)
	if rs1 > 0 {
		if v.registers[rs1].Busy {
			// FIX: Checking the inst.Op == Inst_Store is not a good approach
			// We need this check because the first source is not an ALU input and can't be forwarded
			if inst.Op == Inst_Store || !checkRegisterForwardDecode(v, rs1) {
				return true
			}
		}
	}

	if rs2 > 0 {
		if v.registers[rs2].Busy {
			if !checkRegisterForwardDecode(v, rs2) {
				return true
			}
		}
	}

	return false
}

func (v *Vm) run_fetch() {
	inst := v.program[v.pc]

	{
		n, ok := v._instCycleTable[inst.Op]
		if !ok {
			n = 1 // Default number of cycles in ex is 1
		}
		inst._ex_remaining = n
	}

	// Update the cyle info
	{
		v.cycle_info.Stage_pcs[0] = uint32(v.pc)
	}

	v.pc++

	v._fd_buff[0].inst = inst
	v._fd_buff[0].pc = v.pc
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
		// fmt.Printf("\n\ncycle %d: STALL BEGIN\n", v.Dm.n_cycle)
		// fmt.Printf("Inst: %#v\n", inst)

		return
	} else if v._stall_map&STALL_RAW != 0 {
		v._stall_map &= ^STALL_RAW
		// fmt.Printf("cycle %d: STALL END\n", v.Dm.n_cycle)
	}

	// If this is a branch instruction, stall the fetch until the branch addr is calculated
	if v.isControlInstruction(inst) {
		v._stall_map |= STALL_BRANCH
	}

	// Update the cyle info
	v.cycle_info.Stage_pcs[1] = uint32(pc) - 1

	switch inst._fmt {
	case Fmt_R:
		inst._s1 = v.registers[inst.Rs1].Data
		inst._s2 = v.registers[inst.Rs2].Data

		// Set the destination register as busy
		v.registers[inst.Rd].Busy = true
	case Fmt_I:
		// In load, immediate is placed in a different position so we check it explicitly.
		if inst.Op == Inst_Load {
			inst._imm = inst.Rs1
			inst._s1 = v.registers[inst.Rs2].Data
		} else {
			inst._s1 = v.registers[inst.Rs1].Data
			inst._imm = inst.Rs2
		}

		// Set the destination register as busy
		v.registers[inst.Rd].Busy = true
	case Fmt_S: // sw s1 imm(s2) = mem[rf(s2) + imm] <- s1
		inst._s1 = v.registers[inst.Rd].Data
		inst._imm = inst.Rs1
		inst._s2 = v.registers[inst.Rs2].Data
	case Fmt_B:
		inst._s1 = v.registers[inst.Rd].Data
		inst._s2 = v.registers[inst.Rs1].Data
		inst._imm = inst.Rs2
	case Fmt_U:
		inst._imm = inst.Rs1
		v.registers[inst.Rd].Busy = true
	case Fmt_J:
		inst._imm = inst.Rs1
		v.registers[inst.Rd].Busy = true
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

		rs1, rs2 := getAluInputRegisters(inst)

		s1, t, ok = getRegValueFromBypass(v, rs1)
		if !ok {
			s1 = inst._s1
		}

		// Update cycle info
		v.cycle_info.S1_bypass_status = t

		s2, t, ok = getRegValueFromBypass(v, rs2)
		if !ok {
			s2 = inst._s2
		}

		// Update cycle info
		v.cycle_info.S2_bypass_status = t
	}

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
		inst._result = pc
		v.pc = s1 + inst._imm

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
			v.pc += inst._imm - 1 // v.pc holds the next instruction hence -1
		}
	case Inst_Bne:
		if s1 != s2 {
			v.pc += inst._imm - 1
		}
	case Inst_Blt:
		if s1 < s2 {
			v.pc += inst._imm - 1
		}
	case Inst_Bge:
		if s1 >= s2 {
			v.pc += inst._imm - 1
		}

	case Inst_Jal: // Jump And Link
		inst._result = pc
		v.pc += inst._imm - 1

	case Inst_Lui:
		inst._result = inst._imm
	case Inst_Auipc:
		inst._result = (pc - 1) + inst._imm
	}

	// Branch address is calculated, we can stop stalling.
	if v.isControlInstruction(inst) {
		v._stall_map &= ^STALL_BRANCH
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

	if inst._fmt == Fmt_B {
		return
	}

	// Memory layout is little-endian
	// b3 b2 b1 b0
	switch inst.Op {
	case Inst_Store: // Store word
		u := uint32(inst._s1)
		addr := inst._result
		v.memory[addr+0] = byte(u >> 0)
		v.memory[addr+1] = byte(u >> 8)
		v.memory[addr+2] = byte(u >> 16)
		v.memory[addr+3] = byte(u >> 24)
	case Inst_Load: // Load word
		addr := inst._result
		u := uint32(v.memory[addr+0]) |
			uint32(v.memory[addr+1])<<8 |
			uint32(v.memory[addr+2])<<16 |
			uint32(v.memory[addr+3])<<24

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
		v.registers[inst.Rd].Busy = false

		// We don't allow writes to x0 register
		if inst.Rd == 0 {
			return
		}

		v.registers[inst.Rd].Data = inst._result
	}

}

// NOT USED
// func (v *Vm) RunSequential() {
// 	for !v._halt {
// 		v.run_fetch()
// 		v.run_decode()
// 		v.run_execute()
// 		v.run_memory()
// 		v.run_writeback()
//
// 		v.Dm.n_cycle += 5
// 		v.Dm.n_executed_inst++
// 	}
// }

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
	v.Dm.n_cycle++

	v.cycle_info = Cycle_Info{}

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

	// If no stall, fetch the next instruction
	if uint(v.pc) < v.Dm.program_size && !v._halt && v._stall_map == 0 {
		v.run_fetch()

		v.Dm.n_executed_inst++
	}

	// Save the cycle state
	{
		// For diagnostic purposes
		if v._stall_map > 0 {
			v.Dm.n_stalls++
			v.cycle_info.Stalled = true
		}

		v.Dm.Cycle_infos = append(v.Dm.Cycle_infos, v.cycle_info)
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
	{
		v._fd_buff[1] = v._fd_buff[0]
		v._fd_buff[0].valid = false
	}

	{
		v._dx_buff[1] = v._dx_buff[0]
		v._dx_buff[0].valid = false
	}

	{
		v._xm_buff[1] = v._xm_buff[0]
		v._xm_buff[0].valid = false
	}

	{
		v._mw_buff[1] = v._mw_buff[0]
		v._mw_buff[0].valid = false
	}
}
