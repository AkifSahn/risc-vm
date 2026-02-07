package vm

import (
	"fmt"
	"os"
)

type Dump_Format uint8

const (
	DUMP_BIN = iota
	DUMP_HEX
	DUMP_DEC
)

func (v *Vm) CalculateCpi() float32{
	if !v._halt || v._cycle <= 0{
		fmt.Printf("Run the program first!")
		return -1
	}

	fmt.Printf("size = %d\n", v._program_size)
	return float32(v._cycle)/float32(v._n_executed_inst)
}

func (v *Vm) PrintRegister(reg_str string) {
	reg_num, ok := abiToRegNum[reg_str]
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid register name: '%s'", reg_str)
		return
	}

	reg := v.registers[reg_num]

	status := "free"
	if reg.Busy {
		status = "busy"
	}

	fmt.Printf("%s -> %d (%s)\n", reg_str, reg.Data, status)
}

func (v *Vm) DumpRegisters(format Dump_Format) {
	fmt.Println("------------")
	fmt.Println("Register Dump: ")
	for i, reg := range v.registers {
		status := "free"
		if reg.Busy {
			status = "busy"
		}

		switch format {
		case DUMP_BIN:
			fmt.Printf("\033[0;33m%2d\033[0m = %.32b (%s) (binary)\n", i, reg.Data, status)
		case DUMP_HEX:
			fmt.Printf("\033[0;33m%2d\033[0m = %.4X (%s) (hex)\n", i, reg.Data, status)
		case DUMP_DEC:
			fmt.Printf("\033[0;33m%2d\033[0m = %d (%s) (decimal)\n", i, reg.Data, status)
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
