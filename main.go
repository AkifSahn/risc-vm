package main

import (
	"github.com/AkifSahn/risc-vm/vm"
)

var machine vm.Vm = vm.NewVm()

func main() {
	// machine.LoadProgramFromFile("examples/factorial_recursive.asm")
	machine.LoadProgramFromFile("examples/matmul.asm")
	machine.Run()
	machine.DumpStack(vm.DUMP_DEC)
}
