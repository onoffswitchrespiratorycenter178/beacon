package state

import (
	"context"
	"sync"
)

// Machine coordinates the service registration state machine per RFC 6762 §8.
//
// State flow:
//
//	Initial → Probing → Announcing → Established
//	Probing → ConflictDetected (if conflict)
//
// R001 Decision: Goroutine-per-service architecture
// T037: Implement Machine struct
type Machine struct {
	prober         *Prober
	announcer      *Announcer
	mu             sync.RWMutex
	onStateChange  func(State)
	currentState   State
	injectConflict bool
}

// NewMachine creates a new state machine.
//
// T037: Initialize Machine
func NewMachine() *Machine {
	return &Machine{
		currentState: StateInitial,
		prober:       NewProber(),
		announcer:    NewAnnouncer(),
	}
}

// Run executes the state machine for a service.
//
// RFC 6762 §8: State machine execution
//  1. Probing: Send 3 probes, 250ms apart (~750ms)
//  2. Announcing: Send 2 announcements, 1s apart (~1s)
//  3. Established: Service fully registered
//
// Parameters:
//   - ctx: Context for cancellation
//   - serviceName: Full service name
//
// Returns:
//   - error: Context error if canceled, nil on success
//
// R001: Each service runs in its own goroutine
// T038: Implement Machine.run() with context cancellation
func (sm *Machine) Run(ctx context.Context, serviceName string) error {
	// Transition to Probing
	sm.setState(StateProbing)

	// Phase 1: Probing (~750ms)
	result := sm.prober.Probe(ctx, serviceName)
	if result.Error != nil {
		return result.Error
	}

	if result.Conflict || sm.injectConflict {
		// Conflict detected - stop here
		// Caller (Responder) will handle rename/retry
		sm.setState(StateConflictDetected)
		return nil
	}

	// Transition to Announcing
	sm.setState(StateAnnouncing)

	// Phase 2: Announcing (~1s)
	// Note: Records are built by Responder.Register() via BuildRecordSet()
	// Machine doesn't need records directly - Announcer uses test hooks for unit testing
	// Actual transport integration with records happens in US3 (Response to Queries)
	records := []byte{} // Placeholder for announcer interface
	err := sm.announcer.Announce(ctx, serviceName, records)
	if err != nil {
		return err
	}

	// Transition to Established
	sm.setState(StateEstablished)

	return nil
}

// GetState returns the current state.
//
// T038: State querying
func (sm *Machine) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// setState transitions to a new state.
//
// T038: State transitions with callbacks
func (sm *Machine) setState(newState State) {
	// Manual unlock required: Must release lock before calling user callback to avoid deadlocks.
	// Callback may access state machine, so holding lock would cause deadlock.
	sm.mu.Lock() // nosemgrep: beacon-mutex-defer-unlock
	sm.currentState = newState
	sm.mu.Unlock()

	// Notify test hook (called WITHOUT lock to prevent deadlocks)
	if sm.onStateChange != nil {
		sm.onStateChange(newState)
	}
}

// SetInjectConflict is a test hook to inject conflict during probing.
//
// T062: Test hook for max rename attempts testing
func (sm *Machine) SetInjectConflict(inject bool) {
	sm.injectConflict = inject
}

// GetProber returns the internal Prober for integration with Responder.
//
// US2 GREEN: Allow Responder to access Prober for message capture
func (sm *Machine) GetProber() *Prober {
	return sm.prober
}

// GetAnnouncer returns the internal Announcer for integration with Responder.
//
// US2 GREEN: Allow Responder to access Announcer for message capture
func (sm *Machine) GetAnnouncer() *Announcer {
	return sm.announcer
}
