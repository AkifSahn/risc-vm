package vm

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
		log.Fatalf("Unknown opcode '%s'\n", s)
	}

	return _Inst_Unknown
}

func expandPseudoInstruction(ps Instruction) []Instruction {
	switch ps.Op {
	case Inst_Mv: //  addi rd, rs, 0 Copy register
		return []Instruction{newInstruction(Inst_Addi, ps.Rd, ps.Rs1, 0)}
	case Inst_Not: // xori rd, rs, -1 One’s complement
		return []Instruction{newInstruction(Inst_Xori, ps.Rd, ps.Rs1, -1)}
	case Inst_Neg: // sub rd, x0, rs Two’s complement
		return []Instruction{newInstruction(Inst_Sub, ps.Rd, 0, ps.Rs1)}
	case Inst_Li: // addi Rd x0 imm(Rs1) Load immediate
		return []Instruction{newInstruction(Inst_Addi, ps.Rd, 0, ps.Rs1)}
	case Inst_Jr: // jalr x0, rs, 0 Jump register
		return []Instruction{newInstruction(Inst_Jalr, 0, ps.Rd, 0)}
	case Inst_Ret: // jalr x0, x1, 0 Return from subroutine
		return []Instruction{newInstruction(Inst_Jalr, 0, 1, 0)}
	case Inst_Ble:
		return []Instruction{newInstruction(Inst_Bge, ps.Rs1, ps.Rd, ps.Rs2)}
	case Inst_Bgt:
		return []Instruction{newInstruction(Inst_Blt, ps.Rs1, ps.Rd, ps.Rs2)}
	case Inst_J:
		return []Instruction{newInstruction(Inst_Jal, 0, ps.Rd, 0)}

	/* Based on the risc-v manual, call expands to two instructions:
	call offset:
		auipc x1, offset[31:12]
		jalr x1, x1, offset[11:0]

	And call is used for calling 'far-away' subroutines.

	But in our implementation, we can just expand to 'jal ra offset',
	since it can handle any length jumps
	*/
	case Inst_Call:
		return []Instruction{newInstruction(Inst_Jal, 1, ps.Rd, 0)}
	default:
		// TODO: Better log
		log.Fatalf("ERROR(parser) - Unknown pseudo instruction!")
	}

	return []Instruction{Instruction{}}
}

// Symbol table holding label_str -> line_num
type Symbol_Table map[string]int

var symbol_table = make(Symbol_Table)

var line_num int

func parseRegisterValue(s string) int32 {
	if val, ok := abiToRegNum[s]; ok {
		return int32(val)
	}

	// Check the symbol_table if this is a label we have seen before
	if val, ok := symbol_table[s]; ok {
		return int32(val - line_num)
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("ERR(parser): field is not a valid register, an immediate value or a label name! - '%s'", s)
	}

	return int32(val)
}

func doesLineDeclareLabel(line string) bool {
	if line == "" {
		return false
	}

	tokens := strings.Split(line, ":")

	// Line does not contain ':' sep. Not a label
	if len(tokens) == 1 {
		return false
	}

	return true
}

/*
This function parses the given line into an Instruction object.

returns _, true: if given line is not a valid instruction line
*/
func parseInstructionLine(line string) (Instruction, bool) {
	// Remove parantheses
	line = strings.ReplaceAll(line, "(", " ")
	line = strings.ReplaceAll(line, ")", " ")
	line = strings.ReplaceAll(line, ",", " ")

	var inst Instruction
	tokens := strings.Fields(line)
	n_tokens := len(tokens)

	// Blank line
	if n_tokens == 0 {
		return inst, true
	}

	opcode := tokens[0]
	inst.Op = stringToOpcode(opcode)
	// Only opcode
	if n_tokens == 1 {
		return inst, false
	}

	// read rd
	rd := tokens[1]
	inst.Rd = parseRegisterValue(rd)

	// Only opcode + rd
	if n_tokens == 2 {
		return inst, false
	}

	// read rs1
	rs1 := tokens[2]
	inst.Rs1 = parseRegisterValue(rs1)

	// Only opcode + rd + rs1
	if n_tokens == 3 {
		return inst, false
	}

	// read rs1
	rs2 := tokens[3]
	inst.Rs2 = parseRegisterValue(rs2)

	return inst, false
}

/*
This function tries to parse given file into an array of instructions

Returns 'nil' on failure.
*/
func ParseProgramFromFile(filename string) []Instruction {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("ERR - Failed to open file(%s): %s\n", filename, err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First fill the symbol table for accessing label addresses
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !doesLineDeclareLabel(line) {
			line_num++
			continue
		}

		// Fill the symbol_table
		{
			tokens := strings.Split(line, ":")

			label := strings.TrimSpace(tokens[0])
			symbol_table[label] = line_num

			if strings.TrimSpace(tokens[1]) != "" {
				line_num++
			}
		}
	}

	// Reset the scanner, and line_num
	line_num = 0
	scanner = bufio.NewScanner(file)
	file.Seek(0, 0)

	program := make([]Instruction, 0)
	// Parse the instructions line by line, expand pseudo instructions etc...
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line has a label declaration
		// If line also contains an instruction, remove the label from line
		// If line does not contains instruction, skip the line
		if doesLineDeclareLabel(line) {
			tokens := strings.Split(line, ":")

			if strings.TrimSpace(tokens[1]) != "" {
				line = tokens[1]
			} else {
				continue
			}

		}

		inst, pass := parseInstructionLine(line)
		if pass {
			continue
		}

		if _Inst_Pseudo_start < inst.Op && inst.Op < _Inst_Pseudo_end {
			expanded_inst := expandPseudoInstruction(inst)
			program = append(program, expanded_inst...)
			line_num += len(expanded_inst)
		} else {
			program = append(program, inst)
			line_num++
		}

	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ERR - Failed to read file(%s): %s\n", filename, err.Error())
		return nil
	}

	// TODO: Move the assertion into a seperate assert(x bool) function??
	if len(program) != line_num {
		panic("Assert: len(program) != line_num")
	}

	return program
}
