package rest

import (
	"log"
	"sync"
	"time"

	"github.com/AkifSahn/risc-vm/vm"
)

type Session struct {
	id               string
	last_access_time time.Time
	machine          *vm.Vm
}

var (
	mu       sync.Mutex
	sessions = map[string]*Session{}
)

// Generates a new vm session with the given configs
func newSession(config vm.Vm_Config) (*Session, error) {
	mu.Lock()
	defer mu.Unlock()

	// Generate a session id
	id := generateId()

	vm, err := vm.CreateVm(config)
	if err != nil {
		return nil, err
	}

	session := &Session{id: id, machine: vm, last_access_time: time.Now()}

	// Add to the sessions
	sessions[id] = session

	return session, nil
}

func getSession(id string) *Session {
	s, ok := sessions[id]
	if !ok {
		return nil
	}

	s.last_access_time = time.Now()

	return s
}

const (
	SESSION_MAX_IDLE_MINUTE       = 15 * time.Minute
	SESSION_PURGE_INTERVAL_MINUTE = 5 * time.Minute
)

func purgeIdleSesions() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for _, session := range sessions {
		if now.Sub(session.last_access_time) > SESSION_MAX_IDLE_MINUTE {
			log.Printf("Purged session '%v' for inactivity!\n", session.id)
			delete(sessions, session.id)
		}
	}
}
