package rest

import (
	"sync"

	"github.com/AkifSahn/risc-vm/vm"
)

var (
	mu       sync.Mutex
	sessions = map[string]*vm.Vm{}
)

// Generates a new vm session with the given configs
func newSession(config vm.Vm_Config) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	// Generate a session id
	id := generateId()

	vm, err := vm.CreateVm(config)
	if err != nil {
		return "", err
	}

	// Add to the sessions
	sessions[id] = vm

	return id, nil
}
