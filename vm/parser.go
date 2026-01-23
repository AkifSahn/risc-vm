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

		var inst Instruction
		tokens := strings.Split(line, " ")
		n_tokens := len(tokens)
		if n_tokens == 0{
			continue
		}

		opcode := tokens[0]
		if opcode == ""{
			continue
		}

		inst.Op = stringToOpcode(opcode)
		// Only opcode
		if n_tokens == 1 {
			program = append(program, inst)
			continue
		}

		// read rd
		rd := tokens[1]
		if after, ok := strings.CutPrefix(rd, "R"); ok {
			inst.Rd, err = strconv.Atoi(after)
		} else {
			inst.Rd, err = strconv.Atoi(rd)
		}
		if err != nil {
			log.Fatalf("ERR - Invalid value for rd '%s'\n", rd)
		}

		// Only opcode + rd
		if n_tokens == 2 {
			program = append(program, inst)
			continue
		}

		// read rs1
		rs1 := tokens[2]
		if after, ok := strings.CutPrefix(rs1, "R"); ok {
			inst.Rs1, err = strconv.Atoi(after)
		} else {
			inst.Rs1, err = strconv.Atoi(rs1)
		}
		if err != nil {
			log.Fatalf("ERR - Invalid value for rs1 '%s'\n", rs1)
		}

		// Only opcode + rd + rs1
		if n_tokens == 3 {
			program = append(program, inst)
			continue
		}

		// read rs1
		rs2 := tokens[3]
		if after, ok := strings.CutPrefix(rs2, "R"); ok {
			inst.Rs2, err = strconv.Atoi(after)
		} else {
			inst.Rs2, err = strconv.Atoi(rs2)
		}
		if err != nil {
			log.Fatalf("ERR - Invalid value for rs2 '%s'\n", rs2)
		}

		program = append(program, inst)
	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ERR - Failed to read file(%s): %s\n", filename, err.Error())
		return nil
	}

	return program
}
