package vm

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
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
	Inst_Load
	Inst_Ori
	Inst_Andi
	Inst_Jalr // Jump And Link Reg
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

var abiToRegNum = map[string]int{
	"zero": 0, // zero register
	"ra":   1, // Return address
	"sp":   2, // Stack pointer
	"gp":   3, // Global pointer
	"tp":   4, // Thread pointer

	// Temporaries
	"t0": 5, "t1": 6, "t2": 7,

	// Saved/frame pointer
	"s0": 8, "fp": 8,

	// Saved register
	"s1": 9,

	// Fn args/return values
	"a0": 10, "a1": 11,

	// Fn args
	"a2": 12, "a3": 13,
	"a4": 14, "a5": 15,
	"a6": 16, "a7": 17,

	// Saved registers
	"s2": 18, "s3": 19,
	"s4": 20, "s5": 21,
	"s6": 22, "s7": 23,
	"s8": 24, "s9": 25,
	"s10": 26, "s11": 27,

	// Temporaries
	"t3": 28, "t4": 29,
	"t5": 30, "t6": 31,
}

func stringToOpcode(s string) Inst_Op {
	switch s {
	case "add":
		return Inst_Add
	case "sub":
		return Inst_Sub
	case "mul":
		return Inst_Mul
	case "div":
		return Inst_Div
	case "rem":
		return Inst_Rem
	case "xor":
		return Inst_Xor
	case "or":
		return Inst_Or
	case "and":
		return Inst_And
	case "addi":
		return Inst_Addi
	case "subi":
		return Inst_Subi
	case "xori":
		return Inst_Xori
	case "ori":
		return Inst_Ori
	case "andi":
		return Inst_Andi
	case "jalr":
		return Inst_Jalr
	case "lw":
		return Inst_Load
	case "slli":
		return Inst_Slli
	case "sw":
		return Inst_Store
	case "beq":
		return Inst_Beq
	case "bne":
		return Inst_Bne
	case "blt":
		return Inst_Blt
	case "bge":
		return Inst_Bge
	case "jal":
		return Inst_Jal
	case "lui":
		return Inst_Lui
	case "auipc":
		return Inst_Auipc
	case "mv":
		return Inst_Mv
	case "not":
		return Inst_Not
	case "neg":
		return Inst_Neg
	case "li":
		return Inst_Li
	case "jr":
		return Inst_Jr
	case "ret":
		return Inst_Ret
	case "ble":
		return Inst_Ble
	case "bgt":
		return Inst_Bgt
	case "j":
		return Inst_J
	case "call":
		return Inst_Call
	case "end":
		return Inst_End
	default:
		log.Printf("Unknown opcode '%s'\n", s)
		return _Inst_Unknown
	}
}

type Token struct {
	Pos int
	Val string
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

// Symbol table holding label_str -> line_num
type Symbol_Table map[string]int

var symbol_table = make(Symbol_Table)

var insts_missing_label = make(map[int]string, 0)

var line_num int

func parseRegisterValue(s string) int32 {
	// Register
	if val, ok := abiToRegNum[s]; ok {
		return int32(val)
	}

	// Immediate value
	if val, err := strconv.Atoi(s); err == nil {
		return int32(val)
	}

	// Label
	if val, ok := symbol_table[s]; ok {
		return int32(val - line_num)
	}

	// Label is not in the symbol_table yet
	insts_missing_label[line_num] = s

	return -1
}

func tokenizeLine(line string) []Token {
	var sb strings.Builder

	tokens := make([]Token, 0, 5)
	tok_pos := 0

	appendTok := func() {
		if sb.Len() == 0 {
			return
		}

		tok := Token{Val: sb.String(), Pos: tok_pos}
		tokens = append(tokens, tok)
		tok_pos++

		sb.Reset()
	}

	for _, ch := range line {
		// This is a comment line, stop tokenizing
		if ch == ';' {
			appendTok()
			return tokens
		}

		if unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == ',' || ch == ':' {
			if ch == ':' { // ':' is both seperator and a token
				appendTok()
				sb.WriteRune(':')
				appendTok()
				tok_pos = 0 // If an instruction follows a label declaration, pos should be relative to the opcode
				continue
			}

			appendTok()
			continue
		}

		sb.WriteRune(ch)
	}

	appendTok()
	return tokens
}

// returns error if an invalid token is passed
func fillInstToken(inst *Instruction, tok Token) error {
	switch tok.Pos {
	case 0:
		inst.Op = stringToOpcode(tok.Val)
	case 1:
		inst.Rd = parseRegisterValue(tok.Val)
	case 2:
		inst.Rs1 = parseRegisterValue(tok.Val)
	case 3:
		inst.Rs2 = parseRegisterValue(tok.Val)
	default:
		return fmt.Errorf("Invalid token: %#v\n", tok)
	}

	return nil
}

// Returns list of instructions parsed, the default pc and an error.
func ParseProgramFromFile(filename string) ([]Instruction, int, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, -1, err
	}
	defer file.Close()

	program := make([]Instruction, 0)

	program = append(program, newInstruction(Inst_End, 0, 0, 0))
	line_num = 1

	scanner := bufio.NewScanner(file)

	// First fill the symbol table for accessing label addresses
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		tokens := tokenizeLine(line)
		if len(tokens) >= 2 && tokens[1].Val == ":" {
			// This is a label
			symbol_table[tokens[0].Val] = line_num
			tokens = tokens[2:]
		}

		if len(tokens) <= 0 {
			continue
		}

		var inst Instruction
		for _, tok := range tokens {
			err := fillInstToken(&inst, tok)
			if err != nil {
				return nil, -1, err
			}
		}

		// First expand the pseudo instruction, then determine the format
		if _Inst_Pseudo_start < inst.Op && inst.Op < _Inst_Pseudo_end {
			inst = expandPseudoInstruction(inst)
		}

		inst._fmt = getInstructionFmt(inst)

		program = append(program, inst)
		line_num++
	}

	// If any unresolved label addr, resolve them
	for i, l := range insts_missing_label {
		inst := &program[i]

		target, ok := symbol_table[l]
		if !ok {
			return nil, -1, fmt.Errorf("Undeclared label or illegal label use: '%s'\n", l)
		}

		offset := target - i

		// based on different control instructions, the offset is stored in different place
		switch inst._fmt {
		case Fmt_B:
			inst.Rs2 = int32(offset)
		case Fmt_J:
			inst.Rs1 = int32(offset)
		default:
			return nil, -1, fmt.Errorf("Illegal label use or unknown key: '%s'\n", l)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, -1, err
	}

	// If there is no 'main' lable start from the first instruction
	pc, ok := symbol_table["main"]
	if !ok {
		pc = 1
	}
	return program, pc, nil
}
