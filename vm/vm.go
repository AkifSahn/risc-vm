package vm

import (
	"fmt"
	"log"
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
	Mul
	Div
	Rem
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
	Jalr // Jump And Link Reg
	Load
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
	Ble
	Bgt
	_Pseudo_end

	End
	_Unknown
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

type Pipeline struct {
	f, d, x, m, w Instruction
}

type Pipeline_Buffer struct {
	pc   int32
	inst Instruction
}

const WORD_SIZE = 4              // In bytes
const MEM_SIZE = 100 * WORD_SIZE // 100 Words
const STACK_SIZE = 32            // 32 words

type Vm struct {
	pc       int32
	program  []Instruction
	pipeline Pipeline

	registers [32]Register
	memory    [MEM_SIZE]byte

	_fd_buff Pipeline_Buffer
	_dx_buff Pipeline_Buffer
	_xm_buff Pipeline_Buffer
	_mw_buff Pipeline_Buffer

	_cycle int
	_halt  bool
}

func NewVm() Vm {
	vm := Vm{
		program: make([]Instruction, 0),
	}

	// Initialize stack pointer to the MAX_ADDR
	vm.registers[abiToRegNum["sp"]].Data = MEM_SIZE

	return vm
}

type Dump_Format uint8

const (
	DUMP_BIN = iota
	DUMP_HEX
	DUMP_DEC
)

func (v *Vm) DumpRegisters(format Dump_Format) {
	fmt.Println("------------")
	fmt.Println("Register Dump: ")
	for i, reg := range v.registers {
		switch format {
		case DUMP_BIN:
			fmt.Printf("\033[0;33m%2d\033[0m = %.32b (binary)\n", i, reg.Data)
		case DUMP_HEX:
			fmt.Printf("\033[0;33m%2d\033[0m = %.4X (hex)\n", i, reg.Data)
		case DUMP_DEC:
			fmt.Printf("\033[0;33m%2d\033[0m = %d (decimal)\n", i, reg.Data)
		}
	}
	fmt.Println("------------")
}

func (v *Vm) DumpMemory(start, end int, format Dump_Format) {
	var val int32

	for i := start; i < end; i += 4 {
		val = int32(uint32(v.memory[i]) |
			uint32(v.memory[i+1])<<8 |
			uint32(v.memory[i+2])<<16 |
			uint32(v.memory[i+3])<<24)

		fmt.Printf("\033[0;33m%d\033[0m : ", i)
		switch format {
		case DUMP_BIN:
			fmt.Printf("%.8b %.8b %.8b %.8b", v.memory[i], v.memory[i+1], v.memory[i+2], v.memory[i+3])
			fmt.Printf(" = %.32b (binary)", val)
		case DUMP_HEX:
			fmt.Printf("%.2X %.2X %.2X %.2X", v.memory[i], v.memory[i+1], v.memory[i+2], v.memory[i+3])
			fmt.Printf(" = %.4X (hex)", val)
		case DUMP_DEC:
			fmt.Printf("%3d %3d %3d %3d", v.memory[i], v.memory[i+1], v.memory[i+2], v.memory[i+3])
			fmt.Printf(" = %d (decimal)", val)

		}

		if v.registers[abiToRegNum["sp"]].Data == int32(i) {
			fmt.Print(" <- \033[0;31mSP\033[0m")
		}
		fmt.Println()
	}
}

func (v *Vm) DumpStack(format Dump_Format) {
	fmt.Println("------------")
	fmt.Println("Stack Dump: ")

	s_start := MEM_SIZE - STACK_SIZE
	s_end := MEM_SIZE

	v.DumpMemory(s_start, s_end, format)

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
		// In load, immediate is placed in a different position so we check it explicitly.
		if inst.Op == Load {
			inst._imm = inst.Rs1
			inst._s1 = v.registers[inst.Rs2].Data
		} else {
			inst._s1 = v.registers[inst.Rs1].Data
			inst._imm = inst.Rs2
		}
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
	case Mul:
		inst._result = inst._s1 * inst._s2
	case Div:
		inst._result = inst._s1 / inst._s2
	case Rem:
		inst._result = inst._s1 % inst._s2
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

	case Load: // load word
		addr := inst._s1 + inst._imm
		if addr%(WORD_SIZE) != 0 {
			log.Fatalf("ERROR - Illegal read attempt from unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
		}
		inst._result = addr

	case Jalr:
		inst._result = v._dx_buff.pc
		v.pc = inst._s1 + inst._imm

	case Store: // Store word
		addr := inst._s2 + inst._imm // In bytes

		// Calculated addr is in bytes and WORD_SIZE is in bytes. So convert WORD_SIZE to bits
		if addr%(WORD_SIZE) != 0 {
			log.Fatalf("ERROR - Illegal write attempt to unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
		}

		inst._result = addr // Each memory cell holds one byte

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

	// Memory layout is little-endian
	// b3 b2 b1 b0
	switch inst.Op {
	case Store: // Store word
		u := uint32(inst._s1)
		addr := inst._result
		v.memory[addr+0] = byte(u >> 0)
		v.memory[addr+1] = byte(u >> 8)
		v.memory[addr+2] = byte(u >> 16)
		v.memory[addr+3] = byte(u >> 24)
	case Load: // Load word
		addr := inst._result
		u := uint32(v.memory[addr+0]) |
			uint32(v.memory[addr+1])<<8 |
			uint32(v.memory[addr+2])<<16 |
			uint32(v.memory[addr+3])<<24

		inst._result = int32(u)
	}

	v._mw_buff = v._xm_buff
	v._mw_buff.inst = inst
}

func (v *Vm) Writeback() {
	inst := v._mw_buff.inst

	// We don't want to writeback if instruction type is S
	if inst._type == R || inst._type == I || inst._type == U || inst._type == J {
		// We don't allow writes to x0 register
		if inst.Rd == 0 {
			return
		}

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
