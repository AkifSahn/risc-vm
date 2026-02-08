package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/AkifSahn/risc-vm/vm"
)

type Cmd_Type int

const (
	Cmd_Unknown Cmd_Type = iota
	Cmd_Help
	Cmd_Load
	Cmd_Run
	Cmd_Step
	Cmd_Print
	Cmd_Halt // Halts the vm
	Cmd_Exit
)

type Cmd struct {
	Type   Cmd_Type
	Params []string
}

var cmdToUsage = map[Cmd_Type]string{
	Cmd_Help:  "help [cmd]",
	Cmd_Load:  "load <file_name> - loads the given program to the vm.",
	Cmd_Run:   "run [s|p]- executes the loaded program.",
	Cmd_Step:  "step [n] - executes 'n' cycle. 'n' is 1 by default.",
	Cmd_Print: "print - TODO",
	Cmd_Halt:  "halt - halts the execution of the loaded program.",
	Cmd_Exit:  "exit - Exits the CLI.",
}

func printUsage(t Cmd_Type) {
	read_writer.WriteString("Usage: ")
	read_writer.WriteString(cmdToUsage[t])
	read_writer.WriteRune('\n')
	read_writer.Flush()
}

func printHelp() {
	keys := make([]int, 0, len(cmdToUsage))
	for c := range cmdToUsage {
		keys = append(keys, int(c))
	}

	sort.Ints(keys)

	for _, v := range keys {
		read_writer.WriteString(cmdToUsage[Cmd_Type(v)])
		read_writer.WriteRune('\n')
		read_writer.WriteRune('\n')
	}
	read_writer.Flush()
}

func lookupCmdName(name string) Cmd_Type {
	switch name {
	case "help", "h", "?":
		return Cmd_Help
	case "load":
		return Cmd_Load
	case "run":
		return Cmd_Run
	case "step":
		return Cmd_Step
	case "print", "p":
		return Cmd_Print
	case "halt":
		return Cmd_Halt
	case "exit", "quit", "q":
		return Cmd_Exit
	default:
		fmt.Printf("Unknown command: '%s' \n", name)
		return Cmd_Unknown
	}
}

func parseCmdString(line string) Cmd {
	line = strings.TrimSpace(line)
	if line == "" {
		return Cmd{Type: Cmd_Unknown}
	}

	tokens := strings.Fields(line)
	name := tokens[0]
	params := tokens[1:]

	return Cmd{
		Type:   lookupCmdName(name),
		Params: params,
	}
}

func handleHelp(args []string) {
	if len(args) == 0 {
		printHelp()
		return
	}

	// Only 1 param is allowed
	if len(args) > 1 {
		printUsage(Cmd_Help)
		return
	}

	param_cmd := lookupCmdName(args[0])
	printUsage(param_cmd)
}

func handleLoad(args []string) {
	if len(args) != 1 {
		printUsage(Cmd_Load)
		return
	}

	filename := args[0]
	machine.LoadProgramFromFile(filename)
}

func handleRun(args []string) {
	if len(args) == 0 {
		machine.RunSequential()
		return
	}

	if len(args) > 1 {
		printUsage(Cmd_Run)
		return
	}

	mode := args[0]
	switch mode {
	case "sequential", "s":
		machine.RunSequential()
	case "pipelined", "p":
		machine.RunPipelined()
	default:
		fmt.Fprintf(read_writer, "unknown mode: %s\n", mode)
		return
	}

}

func handlePrint(args []string) {
	if len(args) < 1 || len(args) > 1 {
		printUsage(Cmd_Print)
		return
	}

	mode := args[0]
	switch mode {
	case "reg", "r":
		machine.DumpRegisters(vm.DUMP_DEC)
	case "stack", "s":
		machine.DumpStack(vm.DUMP_DEC)
	default:
		fmt.Fprintf(read_writer, "unknown mode: %s\n", mode)
		return
	}
}

func (cmd Cmd) Handle() {
	switch cmd.Type {
	case Cmd_Help:
		handleHelp(cmd.Params)
	case Cmd_Load:
		handleLoad(cmd.Params)
	case Cmd_Exit:
		return
	case Cmd_Halt:
		printUsage(Cmd_Halt)
	case Cmd_Print:
		handlePrint(cmd.Params)
	case Cmd_Run:
		handleRun(cmd.Params)
	case Cmd_Step:
		printUsage(Cmd_Step)
	case Cmd_Unknown:
	default:
		panic(fmt.Sprintf("unexpected main.Cmd: %#v", cmd))
	}
}

var machine *vm.Vm

var reader = bufio.NewReader(os.Stdin)
var writer = bufio.NewWriter(os.Stdout)
var read_writer *bufio.ReadWriter

const MEM_SIZE = 400
const STACk_SIZE = 200

func main() {
	machine = vm.CreateVm(MEM_SIZE, STACk_SIZE)

	read_writer = bufio.NewReadWriter(reader, writer)
	read_writer.WriteString("--- RISC-V Vm Cli --- ('help')\n")
	read_writer.Flush()

	for {
		read_writer.WriteString("> ")
		read_writer.Flush()

		cmd_str, err := read_writer.ReadString('\n')
		if err != nil {
			panic(fmt.Sprintf("ERROR - Failed to read user input: %s\n", err.Error()))
		}

		// Trim spaces and new-line
		cmd_str = strings.Trim(cmd_str, "\n ")

		cmd := parseCmdString(cmd_str)
		if cmd.Type == Cmd_Exit {
			return
		}

		cmd.Handle()
	}
}
