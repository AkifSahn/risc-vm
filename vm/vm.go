package vm

import (
	"fmt"
)

type Inst_Op int
type Inst_Type int

const (
	R Inst_Type = iota
	I           // immediate
	S           // store
	B           // branch
	U           // Upper immediate
	J
)

const (
	Nop Inst_Op = iota

	_R_start
	Add
	Sub
	Xor
	Or
	And
	_R_end

	_I_start
	Addi
	Subi
	Xori
	Ori
	Andi
	Load // load-word
	Jalr // Jump And Link Reg
	_I_end

	_S_start
	Store // store-word
	_S_end

	_B_start
	Beq
	Bne
	Blt
	Bge
	_B_end

	_J_start
	Jal // Jump And Link
	_J_end

	_U_start
	Lui
	Auipc
	_U_end

	_Pseudo_start
	Mv
	Not
	Neg
	Li
	Jr
	Ret
	_Pseudo_end

	End
	_Unknown
)

type Instruction struct {
	Op  Inst_Op
	Rd  int
	Rs1 int
	Rs2 int

	_s1     int
	_s2     int
	_imm    int
	_result int
	_type   Inst_Type
}

func newInstruction(Op Inst_Op, Rd int, Rs1 int, Rs2 int) Instruction {
	return Instruction{
		Op:  Op,
		Rd:  Rd,
		Rs1: Rs1,
		Rs2: Rs2,
	}
}

type Register struct {
	Data int
	Busy bool
}

type Pipeline struct {
	f, d, x, m, w Instruction
}

type Pipeline_Buffer struct {
	pc   int
	inst Instruction
}

const MEM_SIZE = 1024

type Vm struct {
	pc       int
	program  []Instruction
	pipeline Pipeline

	registers [32]Register
	memory    [MEM_SIZE]int

	_fd_buff Pipeline_Buffer
	_dx_buff Pipeline_Buffer
	_xm_buff Pipeline_Buffer
	_mw_buff Pipeline_Buffer

	_cycle int
	_halt  bool
}

func NewVm() Vm {
	return Vm{
		program: make([]Instruction, 0),
	}
}

func (v *Vm) DumpRegisters() {
	fmt.Println("------------")
	fmt.Println("Register Dump: ")
	for i, reg := range v.registers {
		fmt.Printf("x%d data=%d\n", i, reg.Data)
	}
	fmt.Println("------------")
}

func (v *Vm) DumpAllMemory() {
	fmt.Println("------------")
	fmt.Println("Memory Dump: ")
	for i, data := range v.memory {
		fmt.Printf("(%d)data=%d ", i, data)
		fmt.Println()
	}
	fmt.Println("------------")
}

func (v *Vm) DumpMemory(start, end int) {
	fmt.Println("------------")
	fmt.Println("Memory Dump: ")
	for i := start; i < end; i++ {
		fmt.Printf("(%d)data=%d ", i, v.memory[i])
		fmt.Println()
	}
	fmt.Println("------------")
}

func (v *Vm) LoadProgramFromFile(fileName string) {
	// Parse the file etc...
	program := ParseProgramFromFile(fileName)
	v.SetProgram(program)
}

func (v *Vm) SetProgram(program []Instruction) {
	v.pc = 0
	v.program = program
}

func (v *Vm) Fetch() {
	inst := v.program[v.pc]

	// Determine the instruction type
	if _R_start < inst.Op && inst.Op < _R_end {
		inst._type = R
	} else if _I_start < inst.Op && inst.Op < _I_end {
		inst._type = I
	} else if _S_start < inst.Op && inst.Op < _S_end {
		inst._type = S
	} else if _B_start < inst.Op && inst.Op < _B_end {
		inst._type = B
	} else if _U_start < inst.Op && inst.Op < _U_end {
		inst._type = U
	} else if _J_start < inst.Op && inst.Op < _J_end {
		inst._type = J
	}

	v._fd_buff.inst = inst
	v.pc++

	v._fd_buff.pc = v.pc
}

func (v *Vm) Decode() {
	inst := v._fd_buff.inst

	// TODO: apply double buffer
	// we may want to call an explicit write function for reading/writing from/to registers

	switch inst._type {
	case R:
		inst._s1 = v.registers[inst.Rs1].Data
		inst._s2 = v.registers[inst.Rs2].Data
	case I:
		inst._s1 = v.registers[inst.Rs1].Data
		inst._imm = inst.Rs2
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
	case J:
		inst._imm = inst.Rs1
	}

	v._dx_buff.inst = inst
	v._dx_buff.pc = v._fd_buff.pc
}

func (v *Vm) Execute() {
	inst := v._dx_buff.inst
	switch inst.Op {
	case Add:
		inst._result = inst._s1 + inst._s2
	case Sub:
		inst._result = inst._s1 - inst._s2
	case Xor:
		inst._result = inst._s1 ^ inst._s2
	case Or:
		inst._result = inst._s1 | inst._s2
	case And:
		inst._result = inst._s1 & inst._s2

	case Addi:
		inst._result = inst._s1 + inst._imm
	case Subi:
		inst._result = inst._s1 - inst._imm
	case Xori:
		inst._result = inst._s1 ^ inst._imm
	case Ori:
		inst._result = inst._s1 | inst._imm
	case Andi:
		inst._result = inst._s1 & inst._imm

	case Load: // load
		inst._result = inst._s1 + inst._imm
	case Jalr:
		inst._result = v._dx_buff.pc
		v.pc = inst._s1 + inst._imm

	case Store:
		inst._result = inst._s2 + inst._imm
		fmt.Printf("Storing %d at %d\n", inst._s1, inst._result)

	case Beq:
		if inst._s1 == inst._s2 {
			v.pc += inst._imm - 1 // v.pc holds the next instruction hence -1
		}
	case Bne:
		if inst._s1 != inst._s2 {
			v.pc += inst._imm - 1
		}
	case Blt:
		if inst._s1 < inst._s2 {
			v.pc += inst._imm - 1
		}
	case Bge:
		if inst._s1 >= inst._s2 {
			v.pc += inst._imm - 1
		}

	case Jal: // Jump And Link
		inst._result = v._dx_buff.pc
		v.pc += inst._imm - 1

	case Lui:
		inst._result = inst._imm
	case Auipc:
		inst._result = (v._dx_buff.pc - 1) + inst._imm

	case End:
		v._halt = true
	}

	v._xm_buff.inst = inst
	v._xm_buff.pc = v._dx_buff.pc
}

func (v *Vm) Memory() {
	inst := v._xm_buff.inst
	if inst._type == B {
		return
	}

	switch inst.Op {
	case Store:
		v.memory[inst._result] = inst._s1
	}

	v._mw_buff = v._xm_buff
}

func (v *Vm) Writeback() {
	inst := v._mw_buff.inst

	// We don't want to writeback if instruction type is S
	if inst._type == R || inst._type == I || inst._type == U || inst._type == J {
		v.registers[inst.Rd].Data = inst._result // TODO: apply double buffer
	}
}

func (v *Vm) Run() {
	for !v._halt {
		v.Fetch()
		v.Decode()
		v.Execute()
		v.Memory()
		v.Writeback()
		v._cycle++
	}
}
