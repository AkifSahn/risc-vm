package main

import (
	"fmt"

	"github.com/AkifSahn/risc-vm/vm"
)

const MEM_SIZE = 400
const STACK_SIZE = 200

var machine* vm.Vm = vm.CreateVm(MEM_SIZE, STACK_SIZE)

func main() {
	machine.LoadProgramFromFile("examples/matmul.asm")
	machine.RunSequential()
	machine.DumpStack(vm.DUMP_DEC)
	fmt.Printf("CPI = %f\n", machine.Dm.CalculateCpi())
}
