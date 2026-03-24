package vm

import (
	"fmt"
	"log"
	"sort"
)

type Inst_Op int
type Inst_Fmt int

const (
	Fmt_R Inst_Fmt = iota
	Fmt_I          // immediate
	Fmt_S          // store
	Fmt_B          // branch
	Fmt_U          // Upper immediate
	Fmt_J
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
	Inst_Lw // Load word
	Inst_Lh // Load half
	Inst_Lb // Load byte
	Inst_Ori
	Inst_Andi
	Inst_Jalr // Jump And Link Reg
	Inst_Slli
	_Inst_I_end

	_Inst_S_start
	Inst_Sw // store word
	Inst_Sh // store half
	Inst_Sb // store byte
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

func (inst Instruction) Str() string {
	op := opcodeToStringMap[inst.Op]

	format := getInstructionFmt(inst)
	switch format {
	case Fmt_R: // Reg, reg, reg
		return fmt.Sprintf("%s x%d, x%d, x%d", op, inst.Rd, inst.Rs1, inst.Rs2)
	case Fmt_I: // reg, reg, imm
		return fmt.Sprintf("%s x%d, x%d, %d", op, inst.Rd, inst.Rs1, inst.Rs2)
	case Fmt_S: // reg, imm(reg)
		return fmt.Sprintf("%s x%d, %d(x%d)", op, inst.Rd, inst.Rs1, inst.Rs2)
	case Fmt_B: // reg, reg, imm
		return fmt.Sprintf("%s x%d, x%d, %d", op, inst.Rd, inst.Rs1, inst.Rs2)
	case Fmt_U: // reg, imm
		return fmt.Sprintf("%s x%d, %d", op, inst.Rd, inst.Rs1)
	case Fmt_J: // reg, imm(for branching)
		return fmt.Sprintf("%s x%d, %d", op, inst.Rd, inst.Rs1)
	default:
		panic(fmt.Sprintf("unexpected vm.Inst_Fmt: %#v", format))
	}
}

// @Redundant: mostly same as getAluInputRegisters, find a way to remove one of the one of the functions
func (inst Instruction) getSourceRegisters() (int32, int32) {
	switch inst._fmt {
	case Fmt_R:
		return inst.Rs1, inst.Rs2

	case Fmt_I:
		if inst.isLoad() {
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

func (inst Instruction) getAluInputRegisters() (int32, int32) {
	switch inst._fmt {
	case Fmt_R:
		return inst.Rs1, inst.Rs2

	case Fmt_I:
		if inst.isLoad() {
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

func (inst Instruction) isControlInstruction() bool {
	if inst._fmt == Fmt_B || inst._fmt == Fmt_J || inst.Op == Inst_Jalr {
		return true
	}

	return false
}

func (inst Instruction) isLoad() bool {
	if inst.Op == Inst_Lw || inst.Op == Inst_Lh || inst.Op == Inst_Lb {
		return true
	}

	return false
}

func (inst Instruction) isStore() bool {
	if inst.Op == Inst_Sw || inst.Op == Inst_Sh || inst.Op == Inst_Sb {
		return true
	}

	return false
}

func GetInstructionStringList() []string {
	insts := make([]string, 0, len(opcodeToStringMap))
	for _, str := range opcodeToStringMap {
		insts = append(insts, str)
	}
	sort.Strings(insts)
	return insts
}

func getInstructionFmt(inst Instruction) Inst_Fmt {
	// Determine the instruction type
	if _Inst_R_start < inst.Op && inst.Op < _Inst_R_end {
		return Fmt_R
	} else if _Inst_I_start < inst.Op && inst.Op < _Inst_I_end {
		return Fmt_I
	} else if _Inst_S_start < inst.Op && inst.Op < _Inst_S_end {
		return Fmt_S
	} else if _Inst_B_start < inst.Op && inst.Op < _Inst_B_end {
		return Fmt_B
	} else if _Inst_U_start < inst.Op && inst.Op < _Inst_U_end {
		return Fmt_U
	} else if _Inst_J_start < inst.Op && inst.Op < _Inst_J_end {
		return Fmt_J
	}

	return Fmt_R
}

func expandPseudoInstruction(ps Instruction) Instruction {
	switch ps.Op {
	case Inst_Mv: //  addi rd, rs, 0 Copy register
		return newInstruction(Inst_Addi, ps.Rd, ps.Rs1, 0)
	case Inst_Not: // xori rd, rs, -1 One’s complement
		return newInstruction(Inst_Xori, ps.Rd, ps.Rs1, -1)
	case Inst_Neg: // sub rd, x0, rs Two’s complement
		return newInstruction(Inst_Sub, ps.Rd, 0, ps.Rs1)
	case Inst_Li: // addi Rd x0 imm(Rs1) Load immediate
		return newInstruction(Inst_Addi, ps.Rd, 0, ps.Rs1)
	case Inst_Jr: // jalr x0, rs, 0 Jump register
		return newInstruction(Inst_Jalr, 0, ps.Rd, 0)
	case Inst_Ret: // jalr x0, x1, 0 Return from subroutine
		return newInstruction(Inst_Jalr, 0, 1, 0)
	case Inst_Ble:
		return newInstruction(Inst_Bge, ps.Rs1, ps.Rd, ps.Rs2)
	case Inst_Bgt:
		return newInstruction(Inst_Blt, ps.Rs1, ps.Rd, ps.Rs2)
	case Inst_J:
		return newInstruction(Inst_Jal, 0, ps.Rd, 0)

	/* Based on the risc-v manual, call expands to two instructions:
	call offset:
		auipc x1, offset[31:12]
		jalr x1, x1, offset[11:0]

	And call is used for calling 'far-away' subroutines.

	But in our implementation, we can just expand to 'jal ra offset',
	since it can handle any length jumps
	*/
	case Inst_Call:
		return newInstruction(Inst_Jal, 1, ps.Rd, 0)
	default:
		return ps
	}
}

