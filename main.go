package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/AkifSahn/risc-vm/rest"
	"github.com/AkifSahn/risc-vm/vm"
)

const MEM_SIZE = 400
const STACK_SIZE = 200

func main() {
	serve := flag.Bool("serve", false, "start the REST API server")
	port := flag.String("port", "8080", "server port")
	filename := flag.String("file", "", "assembly file to run")

	flag.Parse()

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

	config, err := vm.CreateConfig(MEM_SIZE, STACK_SIZE, true, true)
	if err != nil {
		fmt.Printf("Configuration error: %s", err.Error())
		os.Exit(1)
	}

	machine, err := vm.CreateVm(*config)
	if err != nil {
		fmt.Printf("Failed to create vm: %s", err.Error())
		os.Exit(1)
	}

	err = machine.LoadProgramFromFile(*filename)
	if err != nil {
		log.Printf("Failed to load program from '%s': %s", *filename, err.Error())
		os.Exit(1)
	}

	machine.RunPipelined()
	machine.DumpRegisters(vm.DUMP_DEC)
	machine.DumpStack(vm.DUMP_DEC)
	machine.Dm.PrintDiagnostics()

	// fmt.Println()
	// fmt.Printf("i\tF\tD\tX\tM\tW\n")
	// for i, info := range machine.Dm.Cycle_infos {
	// 	fmt.Printf("%d:", i)

	// 	for _, p := range info.Stage_pcs {
	// 		if p == 0 {
	// 			fmt.Printf("\t*")
	// 			continue
	// 		}
	// 		fmt.Printf("\t%d", p)
	// 	}

	// 	fmt.Println()
	// }
}
