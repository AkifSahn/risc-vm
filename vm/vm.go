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

	End
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
	_halt bool
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
		fmt.Printf("R%d data=%d", i, reg.Data)
		if reg.Busy {
			fmt.Print(" [busy]")
		} else {
			fmt.Print(" [free]")
		}
		fmt.Println()
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
	}

	v._fd_buff.inst = inst
	v.pc++

	v._fd_buff.pc = v.pc
}

func (v *Vm) Decode() {
	inst := v._fd_buff.inst

	// TODO: apply double buffer
	// we may want to call an explicit write function for reading/writing from/to registers
	inst._s1 = v.registers[inst.Rs1].Data

	switch inst._type {
	case R:
		inst._s2 = v.registers[inst.Rs2].Data
	case I:
		inst._imm = inst.Rs2
	case S:
		inst._s2 = v.registers[inst.Rs2].Data
		inst._imm = inst.Rd
	case B:
		inst._s1 = v.registers[inst.Rd].Data
		inst._s2 = v.registers[inst.Rs1].Data
		inst._imm = inst.Rs2
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

	case Store:
		inst._result = inst._s1 + inst._imm

	case Beq:
		if inst._s1 == inst._s2 {
			v.pc += inst._imm
		}
	case Bne:
		if inst._s1 != inst._s2 {
			v.pc += inst._imm
		}
	case Blt:
		if inst._s1 < inst._s2 {
			v.pc += inst._imm
		}
	case Bge:
		if inst._s1 >= inst._s2 {
			v.pc += inst._imm
		}
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
		v.memory[inst._result] = inst._s2
	}

	v._mw_buff = v._xm_buff
}

func (v *Vm) Writeback() {
	inst := v._mw_buff.inst

	// We don't want to writeback if instruction type is S
	if inst._type == R || inst._type == I {
		v.registers[inst.Rd].Data = inst._result // TODO: apply double buffer
	}
}

func (v *Vm) Run() {
	for ;!v._halt; {
		v.Fetch()
		v.Decode()
		v.Execute()
		v.Memory()
		v.Writeback()
		v._cycle++
	}
}
