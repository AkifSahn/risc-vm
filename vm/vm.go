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

type Pipeline_Buffer struct {
	pc        int32
	inst      Instruction
	_is_empty bool
}

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

const WORD_SIZE = 4              // In bytes
const MEM_SIZE = 100 * WORD_SIZE // 100 Words
const STACK_SIZE = 400           // 32 words

type Vm struct {
	pc            int32
	program       []Instruction
	_program_size int32

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

	vm._fd_buff._is_empty = true
	vm._dx_buff._is_empty = true
	vm._xm_buff._is_empty = true
	vm._mw_buff._is_empty = true

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
	fmt.Printf("Stack Dump: Sp = %d\n", v.registers[abiToRegNum["sp"]].Data)

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
	v._program_size = int32(len(program))
}

func (v *Vm) Fetch() {
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

	v._fd_buff.inst = inst
	v._fd_buff._is_empty = false
	v.pc++

	v._fd_buff.pc = v.pc
}

func (v *Vm) Decode() {
	inst := v._fd_buff.inst
	pc := v._fd_buff.pc
	v._fd_buff._is_empty = true

	// TODO: apply double buffer
	// we may want to call an explicit write function for reading/writing from/to registers

	switch inst._type {
	case R:
		inst._s1 = v.registers[inst.Rs1].Data
		inst._s2 = v.registers[inst.Rs2].Data
	case I:
		// In load, immediate is placed in a different position so we check it explicitly.
		if inst.Op == Inst_Load {
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
	v._dx_buff.pc = pc
	v._dx_buff._is_empty = false
}

func (v *Vm) Execute() {
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
			log.Fatalf("ERROR - Illegal read attempt from unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
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
			log.Fatalf("ERROR - Illegal write attempt to unaligned memory address: '%d'."+
				"Each word adress must be aligned by '%d'", addr, WORD_SIZE)
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

		//Halting at execution stage, breaks the pipeline. So we will halt at the writeback instead
		// case Inst_End:
		// 	v._halt = true
	}

	v._xm_buff.inst = inst
	v._xm_buff.pc = pc
	v._xm_buff._is_empty = false
}

func (v *Vm) Memory() {
	inst := v._xm_buff.inst
	pc := v._xm_buff.pc

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

func (v *Vm) Writeback() {
	inst := v._mw_buff.inst
	// pc := v._mw_buff.pc
	v._mw_buff._is_empty = true

	if inst.Op == Inst_End {
		v._halt = true
		return
	}

	// We don't want to writeback if instruction type is S
	if inst._type == R || inst._type == I || inst._type == U || inst._type == J {
		// We don't allow writes to x0 register
		if inst.Rd == 0 {
			return
		}

		v.registers[inst.Rd].Data = inst._result // TODO: apply double buffer
	}

}

func (v *Vm) RunSequential() {
	for !v._halt {
		v._cycle++
		v.Fetch()
		v.Decode()
		v.Execute()
		v.Memory()
		v.Writeback()
	}
}

func (v *Vm) RunPipelined() {
	for !v._halt {
		v._cycle++
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
	fmt.Printf("At cycle: %d\n", v._cycle)
	if !v._mw_buff._is_empty && !v._halt {
		v.Writeback()
		fmt.Printf("\tinst %d -> Writeback()\n", v._mw_buff.pc)
	}

	if !v._xm_buff._is_empty && !v._halt {
		v.Memory()
		fmt.Printf("\tinst %d -> Memory()\n", v._xm_buff.pc)
	}

	if !v._dx_buff._is_empty && !v._halt {
		v.Execute()
		fmt.Printf("\tinst %d -> Execute()\n", v._dx_buff.pc)
	}

	if !v._fd_buff._is_empty && !v._halt {
		v.Decode()
		fmt.Printf("\tinst %d -> Decode()\n", v._fd_buff.pc)
	}

	if v.pc < v._program_size && !v._halt {
		v.Fetch()
		fmt.Printf("\tinst %d -> Fetch()\n", v.pc)
	}
}
