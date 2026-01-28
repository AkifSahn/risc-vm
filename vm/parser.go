package vm

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func stringToOpcode(s string) Inst_Op {
	switch s {
	case "add":
		return Add
	case "sub":
		return Sub
	case "mul":
		return Mul
	case "div":
		return Div
	case "rem":
		return Rem
	case "xor":
		return Xor
	case "or":
		return Or
	case "and":
		return And
	case "addi":
		return Addi
	case "subi":
		return Subi
	case "xori":
		return Xori
	case "ori":
		return Ori
	case "andi":
		return Andi
	case "lw":
		return Load
	case "jalr":
		return Jalr
	case "sw":
		return Store
	case "beq":
		return Beq
	case "bne":
		return Bne
	case "blt":
		return Blt
	case "bge":
		return Bge
	case "jal":
		return Jal
	case "lui":
		return Lui
	case "auipc":
		return Auipc
	case "mv":
		return Mv
	case "not":
		return Not
	case "neg":
		return Neg
	case "li":
		return Li
	case "jr":
		return Jr
	case "ret":
		return Ret
	case "ble":
		return Ble
	case "bgt":
		return Bgt
	case "end":
		return End
	default:
		log.Fatalf("Unknown opcode '%s'\n", s)
	}

	return _Unknown
}

func expandPseudoInstruction(ps Instruction) Instruction {
	switch ps.Op {
	case Mv: //  addi rd, rs, 0 Copy register
		return newInstruction(Addi, ps.Rd, ps.Rs1, 0)
	case Not: // xori rd, rs, -1 One’s complement
		return newInstruction(Xori, ps.Rd, ps.Rs1, -1)
	case Neg: // sub rd, x0, rs Two’s complement
		return newInstruction(Sub, ps.Rd, 0, ps.Rs1)
	case Li: // addi Rd x0 imm(Rs1) Load immediate
		return newInstruction(Addi, ps.Rd, 0, ps.Rs1)
	case Jr: // jalr x0, rs, 0 Jump register
		return newInstruction(Jalr, 0, ps.Rd, 0)
	case Ret: // jalr x0, x1, 0 Return from subroutine
		return newInstruction(Jalr, 0, 1, 0)
	case Ble:
		return newInstruction(Bge, ps.Rs1, ps.Rd, ps.Rs2)
	case Bgt:
		return newInstruction(Blt, ps.Rs1, ps.Rd, ps.Rs2)
	default:
		// TODO: Better log
		log.Fatalf("ERROR(parser) - Unknown pseudo instruction!")
	}

	return Instruction{}
}

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

		if _Pseudo_start < inst.Op && inst.Op < _Pseudo_end {
			inst = expandPseudoInstruction(inst)
		}

		// We may want to increase the line_num more than 1
		// if the 'expandPseudoInstruction' expands the pseudo instructin into multiple instructions.
		// we can just do 'line_num = len(inst)', assuming inst is an array of instructions.
		line_num++
		program = append(program, inst)
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
