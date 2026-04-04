package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/AkifSahn/risc-vm/rest"
	"github.com/AkifSahn/risc-vm/vm"
)

const MEM_SIZE = 1024
const STACK_SIZE = 200 // We don't really need this

func main() {
	serve := flag.Bool("serve", false, "start the REST API server.")
	port := flag.String("port", "8080", "server port.")

	filename := flag.String("file", "", "assembly file to run.")

	mem_size := flag.Uint("mem", MEM_SIZE, "Simulator memory size in bytes.")

	list_cycles := flag.Bool("list-cycles", false, "List cycle-by-cycle stages.")

	save_test := flag.Bool("make-test", false, "Save the result of the execution as test data.")
	run_tests := flag.Bool("run-tests", false, "Run tests for existing saved test data.")

	dump_memory := flag.Bool("dump-memory", false, "Dump memory to the stdout. Disables the statistics print.")

	flag.Parse()

	if *run_tests {
		err := vm.TestAllExamples()
		if err != nil {
			panic(err.Error())
		}
		return
	}

	if *serve {
		host := "127.0.0.1"

		fmt.Printf("Server started at %s:%s\n", host, *port)
		rest.ListenAndServe(fmt.Sprintf("%s:%s", host, *port))
	} else if *filename == "" {
		fmt.Println("Please provide a filename to simulate!")
		fmt.Println()
		flag.Usage()
		return
	}

	config, err := vm.CreateConfig(uint32(*mem_size), STACK_SIZE, 2, true, true)
	if err != nil {
		fmt.Printf("Configuration error: %s\n", err.Error())
		os.Exit(1)
	}

	machine, err := vm.CreateVm(*config)
	if err != nil {
		fmt.Printf("Failed to create vm: %s\n", err.Error())
		os.Exit(1)
	}

	err = machine.LoadProgramFromFile(*filename)
	if err != nil {
		log.Printf("Failed to load program from '%s': %s\n", *filename, err.Error())
		os.Exit(1)
	}

	machine.RunPipelined()

	if *save_test {
		err := machine.SaveTestState(*filename)
		if err != nil {
			fmt.Printf("Failed to save test state: %s\n", err.Error())
			os.Exit(1)
		}

		return
	}

	if *dump_memory {
		err := binary.Write(os.Stdout, binary.LittleEndian, machine.Memory)
		if err != nil {
			fmt.Printf("Failed to dump memory to STDOUT: %v\n", err.Error())
			os.Exit(1)
		}
		return
	}

	if *list_cycles {
		fmt.Printf("i\tF\tD\tX\tM\tW\n")
		for i, info := range machine.Dm.Cycle_infos {
			fmt.Printf("%d:", i)

			for _, p := range info.Stage_pcs {
				if p == 0 {
					fmt.Printf("\t*")
					continue
				}
				fmt.Printf("\t%d", p)
			}

			fmt.Println()
		}
	}

	machine.DumpRegisters(vm.DUMP_DEC)
	machine.DumpStack(vm.DUMP_DEC)
	machine.Dm.PrintDiagnostics()
}
