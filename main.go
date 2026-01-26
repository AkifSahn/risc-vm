package main

import (
	"github.com/AkifSahn/risc-vm/vm"
)

var machine vm.Vm = vm.NewVm()

func main() {
	// machine.DumpRegisters()
	machine.LoadProgramFromFile("examples/fib.asm")
	machine.Run()
	machine.DumpRegisters()
	machine.DumpMemory(0, 10)
}
