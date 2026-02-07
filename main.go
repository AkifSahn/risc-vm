package main

import (
	"github.com/AkifSahn/risc-vm/vm"
)

var machine vm.Vm = vm.NewVm()

func main() {
	machine.LoadProgramFromFile("examples/pipeline_test.asm")
	machine.RunPipelined()
	machine.DumpStack(vm.DUMP_DEC)
}
