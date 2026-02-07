package vm

import "fmt"

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
