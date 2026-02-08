package vm

import (
	"fmt"
	"log"
)

type Inst_Op int
type Inst_Type int

const (
	STALL_RAW uint8 = 1 << iota
	STALL_BRANCH
)

const (
	R Inst_Type = iota
	I           // immediate
	S           // store
	B           // branch
	U           // Upper immediate
	J
)

const (
	Inst_Nop Inst_Op = iota

	_Inst_R_start
	Inst_Add
	Inst_Sub
	Inst_Mul
	Inst_Div
	Inst_Rem
	Inst_Xor
	Inst_Or
	Inst_And
	_Inst_R_end

	_Inst_I_start
	Inst_Addi
	Inst_Subi
	Inst_Xori
	Inst_Ori
	Inst_Andi
	Inst_Jalr // Jump And Link Reg
	Inst_Load
	Inst_Slli
	_Inst_I_end

	_Inst_S_start
	Inst_Store // store-word
	_Inst_S_end

	_Inst_B_start
	Inst_Beq
	Inst_Bne
	Inst_Blt
	Inst_Bge
	_Inst_B_end

	_Inst_J_start
	Inst_Jal // Jump And Link
	_Inst_J_end

	_Inst_U_start
	Inst_Lui
	Inst_Auipc
	_Inst_U_end

	_Inst_Pseudo_start
	Inst_Mv
	Inst_Not
	Inst_Neg
	Inst_Li
	Inst_Jr
	Inst_Ret
	Inst_Ble
	Inst_Bgt
	Inst_J
	Inst_Call
	_Inst_Pseudo_end

	Inst_End
	_Inst_Unknown
)

type Instruction struct {
	Op  Inst_Op
	Rd  int32
	Rs1 int32
	Rs2 int32

	_s1     int32
	_s2     int32
	_imm    int32
	_result int32
	_type   Inst_Type
}

func newInstruction(Op Inst_Op, Rd int32, Rs1 int32, Rs2 int32) Instruction {
	return Instruction{
		Op:  Op,
		Rd:  Rd,
		Rs1: Rs1,
		Rs2: Rs2,
	}
}

type Register struct {
	Data int32
	Busy bool
}

type Pipeline_Buffer struct {
	pc        int32
	inst      Instruction
	_is_empty bool
}

const WORD_SIZE = 4 // In bytes

type Vm struct {
	pc            int32
	program       []Instruction
	_program_size int32

	registers   [32]Register
	memory      []byte
	_mem_size   int // In bytes
	_stack_size int // In bytes

	_fd_buff Pipeline_Buffer
	_dx_buff Pipeline_Buffer
	_xm_buff Pipeline_Buffer
	_mw_buff Pipeline_Buffer

	_halt  bool
	_stall byte // A bitmap for different stalls?? Is this a good idea??

	Dm Diagnostics_Manager
}

func CreateVm(mem_size, stack_size int) *Vm {
	if mem_size <= 0 || stack_size <= 0 {
		fmt.Println("Memory/stack size must be non-zero!")
		return nil
	}

	if mem_size%WORD_SIZE != 0 {
		fmt.Printf("Invalid memory size '%d', Memory size must be a multiple of word size(%d).",
			mem_size, WORD_SIZE)
		return nil
	}

	if stack_size%WORD_SIZE != 0 {
		fmt.Printf("Invalid stack size '%d', Stack size must be a multiple of word size(%d).",
			mem_size, WORD_SIZE)
		return nil
	}

	if stack_size > mem_size {
		fmt.Printf("Stack(%d) size can not be bigger than memory size(%d).",
			stack_size, mem_size)
		return nil
	}

	vm := Vm{
		program:     make([]Instruction, 0),
		memory:      make([]byte, mem_size),
		Dm:          CreateDiagnosticsManager(),

		_mem_size:   mem_size,
		_stack_size: stack_size,
	}

	// Initialize stack pointer to the MAX_ADDR
	vm.registers[abiToRegNum["sp"]].Data = int32(mem_size)

	vm._fd_buff._is_empty = true
	vm._dx_buff._is_empty = true
	vm._xm_buff._is_empty = true
	vm._mw_buff._is_empty = true

	return &vm
}

func (v *Vm) LoadProgramFromFile(fileName string) {
	// Parse the file etc...
	program := ParseProgramFromFile(fileName)
	v.SetProgram(program)
}

func (v *Vm) SetProgram(program []Instruction) {
	v.pc = 0
	v.program = program
	v._program_size = int32(len(program))
}

func (v *Vm) isControlInstruction(inst Instruction) bool {
	if inst._type == B || inst._type == J || inst.Op == Inst_Jalr {
		return true
	}

	return false
}

// This function checks if we should stall in the decode stage or not based on the instruction.
//
// Checks for RAW(Read After Write) Data Hazard.
func (v *Vm) shouldStallDecode(inst Instruction) bool {
	switch inst._type {
	case R:
		if v.registers[inst.Rs1].Busy || v.registers[inst.Rs2].Busy {
			return true
		}

	case I:
		if inst.Op == Inst_Load {
			if v.registers[inst.Rs2].Busy {
				return true
			}
		} else if v.registers[inst.Rs1].Busy {
			return true
		}

	case S: // sw s1 imm(s2) = mem[rf(s2) + imm] <- s1
		if v.registers[inst.Rd].Busy || v.registers[inst.Rs2].Busy {
			return true
		}

	case B:
		if v.registers[inst.Rd].Busy || v.registers[inst.Rs1].Busy {
			return true
		}

	case U:
		return false

	case J:
		return false
	default:
		panic(fmt.Sprintf("unexpected Inst_Type: %#v", inst._type))
	}

	return false
}

func (v *Vm) run_fetch() {
	inst := v.program[v.pc]

	// Determine the instruction type
	if _Inst_R_start < inst.Op && inst.Op < _Inst_R_end {
		inst._type = R
	} else if _Inst_I_start < inst.Op && inst.Op < _Inst_I_end {
		inst._type = I
	} else if _Inst_S_start < inst.Op && inst.Op < _Inst_S_end {
		inst._type = S
	} else if _Inst_B_start < inst.Op && inst.Op < _Inst_B_end {
		inst._type = B
	} else if _Inst_U_start < inst.Op && inst.Op < _Inst_U_end {
		inst._type = U
	} else if _Inst_J_start < inst.Op && inst.Op < _Inst_J_end {
		inst._type = J
	}

	// Check for Control hazards
	if v.isControlInstruction(inst) {
		v._stall |= STALL_BRANCH
	}

	v._fd_buff.inst = inst
	v._fd_buff._is_empty = false
	v.pc++

	v._fd_buff.pc = v.pc
}

func (v *Vm) run_decode() {
	inst := v._fd_buff.inst
	pc := v._fd_buff.pc
	v._fd_buff._is_empty = true

	if v.shouldStallDecode(inst) {
		v._stall |= STALL_RAW
		v._fd_buff._is_empty = false
		return
	} else if v._stall&STALL_RAW != 0 {
		v._stall &= ^STALL_RAW
	}

	switch inst._type {
	case R:
		inst._s1 = v.registers[inst.Rs1].Data
		inst._s2 = v.registers[inst.Rs2].Data

		// Set the destination register as busy
		v.registers[inst.Rd].Busy = true
	case I:
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
	case S: // sw s1 imm(s2) = mem[rf(s2) + imm] <- s1
		inst._s1 = v.registers[inst.Rd].Data
		inst._imm = inst.Rs1
		inst._s2 = v.registers[inst.Rs2].Data
	case B:
		inst._s1 = v.registers[inst.Rd].Data
		inst._s2 = v.registers[inst.Rs1].Data
		inst._imm = inst.Rs2
	case U:
		inst._imm = inst.Rs1
		v.registers[inst.Rd].Busy = true
	case J:
		inst._imm = inst.Rs1
		v.registers[inst.Rd].Busy = true
	}

	v._dx_buff.inst = inst
	v._dx_buff.pc = pc
	v._dx_buff._is_empty = false
}

func (v *Vm) run_execute() {
	inst := v._dx_buff.inst
	pc := v._dx_buff.pc
	v._dx_buff._is_empty = true

	switch inst.Op {
	case Inst_Add:
		inst._result = inst._s1 + inst._s2
	case Inst_Sub:
		inst._result = inst._s1 - inst._s2
	case Inst_Mul:
		inst._result = inst._s1 * inst._s2
	case Inst_Div:
		inst._result = inst._s1 / inst._s2
	case Inst_Rem:
		inst._result = inst._s1 % inst._s2
	case Inst_Xor:
		inst._result = inst._s1 ^ inst._s2
	case Inst_Or:
		inst._result = inst._s1 | inst._s2
	case Inst_And:
		inst._result = inst._s1 & inst._s2

	case Inst_Addi:
		inst._result = inst._s1 + inst._imm
	case Inst_Subi:
		inst._result = inst._s1 - inst._imm
	case Inst_Xori:
		inst._result = inst._s1 ^ inst._imm
	case Inst_Ori:
		inst._result = inst._s1 | inst._imm
	case Inst_Andi:
		inst._result = inst._s1 & inst._imm

	case Inst_Load: // load word
		addr := inst._s1 + inst._imm
		if addr%(WORD_SIZE) != 0 {
			log.Printf("ERROR - Illegal read attempt from unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
			v._halt = true
		}
		inst._result = addr

	case Inst_Jalr:
		inst._result = pc
		v.pc = inst._s1 + inst._imm

	case Inst_Slli: // rd = rs1 << imm[0:4]
		inst._result = inst._s1 << inst._imm

	case Inst_Store: // Store word
		addr := inst._s2 + inst._imm // In bytes

		// Calculated addr is in bytes and WORD_SIZE is in bytes. So convert WORD_SIZE to bits
		if addr%(WORD_SIZE) != 0 {
			log.Printf("ERROR - Illegal write attempt to unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
			v._halt = true
		}

		inst._result = addr // Each memory cell holds one byte

	case Inst_Beq:
		if inst._s1 == inst._s2 {
			v.pc += inst._imm - 1 // v.pc holds the next instruction hence -1
		}
	case Inst_Bne:
		if inst._s1 != inst._s2 {
			v.pc += inst._imm - 1
		}
	case Inst_Blt:
		if inst._s1 < inst._s2 {
			v.pc += inst._imm - 1
		}
	case Inst_Bge:
		if inst._s1 >= inst._s2 {
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

	// If this is a control instruction, we can stop the stall in this stage
	// since we calculated the target pc
	if v.isControlInstruction(inst) {
		v._stall &= ^STALL_BRANCH
	}

	v._xm_buff.inst = inst
	v._xm_buff.pc = pc
	v._xm_buff._is_empty = false
}

func (v *Vm) run_memory() {
	inst := v._xm_buff.inst
	pc := v._xm_buff.pc
	v._xm_buff._is_empty = true

	if inst._type == B {
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

	// TODO: should we handle B-type differently??
	v._mw_buff.inst = inst
	v._mw_buff.pc = pc
	v._mw_buff._is_empty = false
}

func (v *Vm) run_writeback() {
	inst := v._mw_buff.inst
	v._mw_buff._is_empty = true

	if inst.Op == Inst_End {
		v._halt = true
		return
	}

	// We don't want to writeback if instruction type is S
	if inst._type == R || inst._type == I || inst._type == U || inst._type == J {
		// set the destination register as free
		v.registers[inst.Rd].Busy = false

		// We don't allow writes to x0 register
		if inst.Rd == 0 {
			return
		}

		v.registers[inst.Rd].Data = inst._result
	}

}

func (v *Vm) RunSequential() {
	for !v._halt {
		v.run_fetch()
		v.run_decode()
		v.run_execute()
		v.run_memory()
		v.run_writeback()

		v.Dm.n_cycle += 5
		v.Dm.n_executed_inst++
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
	v.Dm.n_cycle++
	if !v._mw_buff._is_empty && !v._halt {
		v.run_writeback()
	}

	if !v._xm_buff._is_empty && !v._halt {
		v.run_memory()
	}

	if !v._dx_buff._is_empty && !v._halt {
		v.run_execute()
	}

	// For now, only data hazard we can see is a RAW data hazard
	// which requires stall insted of decoding. So if we signal a stall in the decode
	// and don't fetch next instruction, in the next cycle we will try to decode the same instruction again
	// since we are getting the instruction from the pipeline buffers
	if !v._fd_buff._is_empty && !v._halt {
		v.run_decode()
	}

	if v.pc < v._program_size && !v._halt && v._stall == 0 {
		v.run_fetch()
		v.Dm.n_executed_inst++
	}
}
