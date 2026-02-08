package main

import (
	"fmt"

	"github.com/AkifSahn/risc-vm/vm"
)

var machine vm.Vm = vm.CreateVm()

func main() {
	machine.LoadProgramFromFile("examples/matmul.asm")
	machine.RunSequential()
	machine.DumpStack(vm.DUMP_DEC)
	fmt.Printf("CPI = %f\n", machine.Dm.CalculateCpi())
}
