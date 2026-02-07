package main

import (
	"github.com/AkifSahn/risc-vm/vm"
)

var machine vm.Vm = vm.CreateVm()

func main() {
	machine.LoadProgramFromFile("examples/matmul.asm")
	machine.RunPipelined()
	machine.DumpStack(vm.DUMP_DEC)
}
