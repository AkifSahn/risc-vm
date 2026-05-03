package vm

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Saved_State struct {
	Config    Vm_Config
	Registers [32]Register
	Memory    []byte
}

const (
	SAVE_FOLDER   = "tests"
	SOURCE_FOLDER = "examples"
	SAVE_SUFFIX   = ".saved"
)

func (v Vm) SaveTestState(src_path string) error {
	var state Saved_State

	state.Config = v.Config
	state.Registers = v.Registers
	state.Memory = make([]byte, v.Config.Mem_size)
	copy(state.Memory, v.Memory)

	file_path := fmt.Sprintf("./%s/%s%s", SAVE_FOLDER, filepath.Base(src_path), SAVE_SUFFIX)
	f, err := os.Create(file_path)
	if err != nil {
		return fmt.Errorf("Failed to create file '%v': %v\n", file_path, err.Error())
	}
	defer f.Close()

	err = binary.Write(f, binary.LittleEndian, state.Config)
	if err != nil {
		return fmt.Errorf("Failed to write 'state.config' to saved state '%s': %s\n", file_path, err.Error())
	}

	err = binary.Write(f, binary.LittleEndian, state.Registers)
	if err != nil {
		return fmt.Errorf("Failed to write 'state.registers' to saved state '%s': %s\n", file_path, err.Error())
	}

	_, err = f.Write(state.Memory)
	if err != nil {
		return fmt.Errorf("Failed to write 'state.memory' to saved state '%s': %s\n", file_path, err.Error())
	}

	return nil
}

func TestAllExamples() error {
	err := filepath.WalkDir(SAVE_FOLDER, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		src_name, found := strings.CutSuffix(filepath.Base(path), fmt.Sprintf("%s", SAVE_SUFFIX))
		if !found {
			fmt.Printf("Ignoring '%v', file does not have a '%v' suffix.\n", path, SAVE_SUFFIX)
			return nil
		}

		var saved_state Saved_State
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		err = binary.Read(f, binary.LittleEndian, &saved_state.Config)
		if err != nil {
			fmt.Printf("Failed to read 'saved_state.config' from '%v': %v\n", path, err.Error())
			return nil
		}

		err = binary.Read(f, binary.LittleEndian, &saved_state.Registers)
		if err != nil {
			fmt.Printf("Failed to read 'saved_state.registers' from '%v': %v\n", path, err.Error())
			return nil
		}

		saved_state.Memory = make([]byte, saved_state.Config.Mem_size)
		_, err = f.Read(saved_state.Memory)
		if err != nil {
			fmt.Printf("Failed to read 'saved_state.memory' from '%v': %v\n", path, err.Error())
			return nil
		}

		f.Close()

		// Different combinations for forwarding and Branch prediction configs
		forwardingAndBpConfigs := [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}}

		fmt.Printf("'%v:'\n", path)
		// Create a vm and run the example code on it
		for _, combination := range forwardingAndBpConfigs {
			failed := false;

			forwarding, bp := combination[0], combination[1]
			cfg, _ := CreateConfig(saved_state.Config.Mem_size, saved_state.Config.Stack_size, saved_state.Config.Bp_nbit, forwarding, bp)
			vm, err := CreateVm(*cfg)
			if err != nil {
				return err
			}

			err = vm.LoadProgramFromFile(fmt.Sprintf("%s/%s", SOURCE_FOLDER, src_name))
			if err != nil {
				fmt.Printf("\tFailed to load program '%v' for testing. '%v'\n", src_name, err.Error())
				return nil
			}

			vm.RunPipelined()

			var memoryErrs []string
			var registerErrs []string
			if slices.Compare(vm.Memory, saved_state.Memory) != 0 {
				memoryErrs = append(memoryErrs, "Memories does not match")
			}

			for i, reg := range vm.Registers {
				if saved_state.Registers[i].Data != reg.Data || saved_state.Registers[i].Busy != reg.Busy {
					msg := fmt.Sprintf("0x%v %v != %v", i, reg, saved_state.Registers[i])
					registerErrs = append(registerErrs, msg)
				}
			}

			if len(memoryErrs) > 0 || len(registerErrs) > 0 {
				fmt.Printf("\t\033[0;31mFAIL\033[0m 'Forwarding: %v, BP: %v'\n", forwarding, bp)
				failed = true
			}

			if len(memoryErrs) > 0 {
				fmt.Printf("\tMemories does not match [\n")
				for i, err := range memoryErrs {
					fmt.Printf("\t\t'%v'", err)
					if i < len(memoryErrs)-1 {
						fmt.Printf(", ")
					}
				}
				fmt.Printf("\n\t]\n")
			}

			if len(registerErrs) > 0 {
				fmt.Printf("\tRegisters does not match [\n")
				for i, err := range registerErrs {
					fmt.Printf("\t\t'%v'", err)
					if i < len(registerErrs)-1 {
						fmt.Printf("\n")
					}
				}
				fmt.Printf("\n\t]\n")
			}

			if !failed {
				fmt.Printf("\t\033[0;32mPASS\033[0m 'Forwarding: %v, BP: %v'\n", forwarding, bp)
			}
		}
		fmt.Println("================================")
		return err
	})

	if err != nil {
		return err
	}
	return nil
}
