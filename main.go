package main

import (
	"fmt"
	"log"
	"os"

	"github.com/AkifSahn/risc-vm/vm"
)

const MEM_SIZE = 400
const STACK_SIZE = 200


func main() {
	machine, err := vm.CreateVm(MEM_SIZE, STACK_SIZE)
	if err != nil {
		fmt.Printf("Failed to create vm: %s", err.Error())
		os.Exit(1)
	}

	filename := "examples/pipeline_test.asm"
	err = machine.LoadProgramFromFile(filename)
	if err != nil {
		log.Printf("Failed to load program from '%s': %s", filename, err.Error())
		os.Exit(1)
	}

	machine.RunPipelined()
	machine.DumpRegisters(vm.DUMP_DEC)
	machine.DumpStack(vm.DUMP_DEC)
	machine.Dm.PrintDiagnostics()
}
