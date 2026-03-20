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

type bypass_type uint8

const (
	BYPASS_XM bypass_type = iota + 1 // Bypass from IX/MEM pipeline buffer
	BYPASS_MW                        // Bypass from MEM/WB pipeline buffer
)

type Cycle_Info struct {
	Stage_pcs        [5]uint32   `json:"stages"`
	Stalled          bool        `json:"stalled"`
	S1_bypass_status bypass_type `json:"s1_bypass"`
	S2_bypass_status bypass_type `json:"s2_bypass"`
}

type Diagnostics_Manager struct {
	Program_size    uint
	N_executed_inst uint
	N_cycle         uint
	N_stalls        uint
	N_forwards      uint
	N_branch        uint
	N_mispred       uint

	Cycle_infos []Cycle_Info

	Bp_enabled         bool
	Forwarding_enabled bool
}

func CreateDiagnosticsManager() Diagnostics_Manager {
	return Diagnostics_Manager{}
}

func (dm *Diagnostics_Manager) CalculateCpi() float32 {
	if dm.N_cycle <= 0 && dm.N_executed_inst <= 0 {
		return -1
	}

	return float32(dm.N_cycle) / float32(dm.N_executed_inst)
}

func (dm *Diagnostics_Manager) PrintDiagnostics() {
	fmt.Println("--- Diagnostics ---")
	fmt.Printf("%-30s %d\n", "Program size:", dm.Program_size)
	fmt.Printf("%-30s %.2f\n", "CPI:", dm.CalculateCpi())
	fmt.Printf("%-30s %d\n", "instructions executed:", dm.N_executed_inst)
	fmt.Printf("%-30s %d\n", "cycles:", dm.N_cycle)
	fmt.Printf("%-30s %d\n", "stalls:", dm.N_stalls)
	fmt.Printf("%-30s %d\n", "forwards:", dm.N_forwards)

	if dm.Bp_enabled {
		if dm.N_branch <= 0 {
			fmt.Printf("%-30s %s\n", "prediction accuracy:", "No branches")
		} else {
			fmt.Printf("%-30s %d%%\n", "prediction accuracy:", 100*(dm.N_branch-dm.N_mispred)/dm.N_branch)
		}
	} else {
		fmt.Printf("%-30s %s\n", "prediction accuracy:", "-")
	}

	fmt.Println()

	fmt.Printf("%-30s %#v\n", "forwarding:", dm.Forwarding_enabled)
	fmt.Printf("%-30s %#v\n", "branch prediction:", dm.Bp_enabled)
	fmt.Println("--- end of diagnostics ---")
}

func (v *Vm) PrintRegister(reg_str string) {
	reg_num, ok := abiToRegNum[reg_str]
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid register name: '%s'", reg_str)
		return
	}

	reg := v.Registers[reg_num]

	status := "free"
	if reg.Busy > 0 {
		status = "busy"
	}

	fmt.Printf("%s -> %d (%s)\n", reg_str, reg.Data, status)
}

func (v *Vm) DumpRegisters(format Dump_Format) {
	fmt.Println("------------")
	fmt.Println("Register Dump: ")
	for i, reg := range v.Registers {
		status := "free"
		if reg.Busy > 0 {
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
		val = int32(uint32(v.Memory[i]) |
			uint32(v.Memory[i+1])<<8 |
			uint32(v.Memory[i+2])<<16 |
			uint32(v.Memory[i+3])<<24)

		fmt.Printf("\033[0;33m%d\033[0m : ", i)
		switch format {
		case DUMP_BIN:
			fmt.Printf("%.8b %.8b %.8b %.8b", v.Memory[i], v.Memory[i+1], v.Memory[i+2], v.Memory[i+3])
			fmt.Printf(" = %.32b (binary)", val)
		case DUMP_HEX:
			fmt.Printf("%.2X %.2X %.2X %.2X", v.Memory[i], v.Memory[i+1], v.Memory[i+2], v.Memory[i+3])
			fmt.Printf(" = %.4X (hex)", val)
		case DUMP_DEC:
			fmt.Printf("%3d %3d %3d %3d", v.Memory[i], v.Memory[i+1], v.Memory[i+2], v.Memory[i+3])
			fmt.Printf(" = %d (decimal)", val)

		}

		if v.Registers[abiToRegNum["sp"]].Data == int32(i) {
			fmt.Print(" <- \033[0;31mSP\033[0m")
		}
		fmt.Println()
	}
}

func (v *Vm) DumpStack(format Dump_Format) {
	fmt.Println("------------")
	fmt.Printf("Stack Dump: Sp = %d\n", v.Registers[abiToRegNum["sp"]].Data)

	s_start := v.Config.mem_size - v.Config.stack_size
	s_end := v.Config.mem_size

	v.DumpMemory(s_start, s_end, format)

	fmt.Println("------------")
}

