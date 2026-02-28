package vm

import "log"

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

