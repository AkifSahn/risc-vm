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
	case "load":
		return Load
	case "store":
		return Store
	case "beq":
		return Beq
	case "bne":
		return Bne
	case "blt":
		return Blt
	case "bge":
		return Bge
	case "end":
		return End
	default:
		log.Fatalf("Unknown opcode '%s'\n", s)
	}

	return End
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

func parseRegisterValue(s string) int {
	if reg, ok := abiToRegNum[s]; ok {
		return reg
	}

	reg, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("ERR(parser): field is not a valid register nor an immediate value! - '%s'", s)
	}

	return reg
}

/*
This function parses the given line into an Instruction object.

If given line is not a valid instruction line returns _, true
*/
func parseInstructionLine(line string) (Instruction, bool) {
	// Remove parantheses
	line = strings.ReplaceAll(line, "(", "")
	line = strings.ReplaceAll(line, ")", "")

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
This function tries to parse given file into an array of instructions struct

Returns 'nil' on failure.
*/
func ParseProgramFromFile(filename string) []Instruction {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("ERR - Failed to open file(%s): %s\n", filename, err.Error())
	}
	defer file.Close()

	program := make([]Instruction, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		inst, pass := parseInstructionLine(line)
		if pass {
			continue
		}

		program = append(program, inst)
	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ERR - Failed to read file(%s): %s\n", filename, err.Error())
		return nil
	}

	return program
}
