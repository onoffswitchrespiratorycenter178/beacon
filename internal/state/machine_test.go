package state

import (
	"context"
	"testing"
	"time"
)

// TestMachine_Transitions_RED tests state machine transitions per RFC 6762 §8.
//
// TDD Phase: RED - These tests will FAIL until we implement Machine
//
// RFC 6762 §8: State Transitions
//   - Initial → Probing: Start probing when service registered
//   - Probing → Announcing: After successful probing (no conflicts)
//   - Announcing → Established: After announcements sent
//   - Probing → Initial: On conflict detected (for retry/rename)
//
// R001 Decision: Use goroutine-per-service architecture
// FR-029: System MUST implement correct state transitions
// T026: Write state machine tests
func TestMachine_Run_Probing_To_Announcing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	ctx := context.Background()
	machine := NewMachine()

	// Track state transitions
	var states []State
	machine.onStateChange = func(newState State) {
		states = append(states, newState)
	}

	// Run state machine in goroutine
	done := make(chan error, 1)
	go func() {
		done <- machine.Run(ctx, testServiceName)
	}()

	// Wait for state machine to complete (~1.5s: 500ms probing + 1s announcing)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not complete within 3 seconds")
	}

	// Verify state transitions: Initial → Probing → Announcing → Established
	wantStates := []State{StateProbing, StateAnnouncing, StateEstablished}
	if len(states) != len(wantStates) {
		t.Fatalf("state transitions = %v, want %v", states, wantStates)
	}

	for i, want := range wantStates {
		if states[i] != want {
			t.Errorf("state[%d] = %v, want %v", i, states[i], want)
		}
	}
}

// TestMachine_Run_ConflictHandling_RED tests conflict handling during probing.
//
// TDD Phase: RED
//
// RFC 6762 §8.1: If conflict detected during probing, abort and signal caller
// T026: Test state machine handles conflicts
func TestMachine_Run_ConflictDetected(t *testing.T) {
	ctx := context.Background()
	machine := NewMachine()

	// Inject conflict during probing
	machine.injectConflict = true

	// Run state machine
	err := machine.Run(ctx, testServiceName)

	// Conflict is not an error - state machine completes normally
	// Caller (Responder) will check for ConflictDetected state
	if err != nil {
		t.Errorf("Run() error = %v, want nil (conflict should be signaled via state, not error)", err)
	}

	// Verify state machine stopped at Probing (did not progress to Announcing)
	if machine.currentState == StateAnnouncing || machine.currentState == StateEstablished {
		t.Errorf("currentState = %v, want StateProbing (stopped at conflict)", machine.currentState)
	}
}

// TestMachine_Run_StateConflictDetected_Exists tests that StateConflictDetected exists and is reachable.
//
// TDD Phase: GREEN (already passing - state exists from T038)
//
// RFC 6762 §8.1: If conflict detected during probing, transition to ConflictDetected state
//
// T060: Verify StateConflictDetected state transition exists
func TestMachine_Run_StateConflictDetected_Exists(t *testing.T) {
	ctx := context.Background()
	machine := NewMachine()

	// Track state transitions
	var states []State
	machine.onStateChange = func(newState State) {
		states = append(states, newState)
	}

	// Inject conflict during probing
	machine.SetInjectConflict(true)

	// Run state machine
	err := machine.Run(ctx, testServiceName)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	// Verify state transitions: Initial → Probing → ConflictDetected
	wantStates := []State{StateProbing, StateConflictDetected}
	if len(states) != len(wantStates) {
		t.Fatalf("state transitions = %v, want %v", states, wantStates)
	}

	for i, want := range wantStates {
		if states[i] != want {
			t.Errorf("state[%d] = %v, want %v", i, states[i], want)
		}
	}

	// Verify final state is ConflictDetected
	finalState := machine.GetState()
	if finalState != StateConflictDetected {
		t.Errorf("GetState() = %v, want StateConflictDetected", finalState)
	}
}

// TestMachine_Run_Cancellation_RED tests context cancellation.
//
// TDD Phase: RED
//
// FR-009: State machine MUST respect context cancellation
// T026: Test state machine cancellation
func TestMachine_Run_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	machine := NewMachine()

	// Cancel context after 500ms (during probing or announcing)
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := machine.Run(ctx, testServiceName)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Run() error = nil, want context.Canceled")
	}

	// Should abort quickly after cancellation (~500ms, not full 1.75s)
	if elapsed > 800*time.Millisecond {
		t.Errorf("Run() took %v after cancellation, want <800ms", elapsed)
	}
}

// TestMachine_GetState_RED tests querying current state.
//
// TDD Phase: RED
//
// FR-029: Applications MUST be able to query service state
// T026: Test GetState() method
func TestMachine_GetState(t *testing.T) {
	machine := NewMachine()

	// Initial state
	if state := machine.GetState(); state != StateInitial {
		t.Errorf("GetState() = %v, want StateInitial", state)
	}

	// Manually transition to Probing (for testing)
	machine.setState(StateProbing)

	if state := machine.GetState(); state != StateProbing {
		t.Errorf("GetState() = %v, want StateProbing after setState", state)
	}
}

// TestMachine_Run_TimingAccuracy_RED tests timing accuracy of state transitions.
//
// TDD Phase: RED
//
// RFC 6762 §8: Timing requirements
//   - Probing: ~500ms (2 intervals of 250ms between 3 probes)
//   - Announcing: ~1s (2 announcements × 1s)
//   - Total: ~1.75s
//
// T026: Test timing accuracy
func TestMachine_Run_TimingAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	ctx := context.Background()
	machine := NewMachine()

	// Track state transition times
	var probingStart, announcingStart, establishedStart time.Time
	machine.onStateChange = func(newState State) {
		switch newState {
		case StateProbing:
			probingStart = time.Now()
		case StateAnnouncing:
			announcingStart = time.Now()
		case StateEstablished:
			establishedStart = time.Now()
		}
	}

	start := time.Now()
	err := machine.Run(ctx, testServiceName)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	// Check total duration (~1.75s ± 300ms)
	totalElapsed := time.Since(start)
	if totalElapsed < 1450*time.Millisecond || totalElapsed > 2050*time.Millisecond {
		t.Errorf("Total duration = %v, want ~1.75s (range: 1.45s-2.05s)", totalElapsed)
	}

	// Check probing duration (~500ms ± 200ms)
	if !probingStart.IsZero() && !announcingStart.IsZero() {
		probingDuration := announcingStart.Sub(probingStart)
		if probingDuration < 350*time.Millisecond || probingDuration > 650*time.Millisecond {
			t.Errorf("Probing duration = %v, want ~500ms (range: 350ms-650ms)", probingDuration)
		}
	}

	// Check announcing duration (~1s ± 200ms)
	if !announcingStart.IsZero() && !establishedStart.IsZero() {
		announcingDuration := establishedStart.Sub(announcingStart)
		if announcingDuration < 800*time.Millisecond || announcingDuration > 1200*time.Millisecond {
			t.Errorf("Announcing duration = %v, want ~1s (range: 800ms-1.2s)", announcingDuration)
		}
	}
}

// TestMachine_ConcurrentRun_RED tests running multiple state machines concurrently.
//
// TDD Phase: RED
//
// R001 Decision: Goroutine-per-service architecture
// FR-030: System MUST support concurrent state machines
// T026: Test concurrent state machines
func TestMachine_ConcurrentRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	ctx := context.Background()

	// Run 10 state machines concurrently
	numMachines := 10
	done := make(chan error, numMachines)

	for i := 0; i < numMachines; i++ {
		go func(id int) {
			machine := NewMachine()
			serviceName := "Service-" + string(rune('0'+id)) + "._http._tcp.local"
			done <- machine.Run(ctx, serviceName)
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numMachines; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("concurrent Run() error = %v, want nil", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent Run() did not complete within 5 seconds")
		}
	}
}
