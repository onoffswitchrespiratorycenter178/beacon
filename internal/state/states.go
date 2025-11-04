// Package state implements the mDNS service registration state machine per RFC 6762 §8.
package state

// State represents the current state of the service registration state machine.
//
// RFC 6762 §8: State transitions
//   - Initial: Service registered, not yet probing
//   - Probing: Sending probe queries to detect conflicts
//   - Announcing: Broadcasting service announcements
//   - Established: Service fully registered and discoverable
//   - ConflictDetected: Naming conflict detected during probing
//
// T037: Define State type
type State int

const (
	// StateInitial is the starting state when a service is first registered.
	StateInitial State = iota

	// StateProbing indicates the service is actively probing for conflicts.
	// RFC 6762 §8.1: Send 3 probe queries, 250ms apart (~750ms total)
	StateProbing

	// StateAnnouncing indicates the service is broadcasting announcements.
	// RFC 6762 §8.3: Send 2 announcements, 1s apart (~1s total)
	StateAnnouncing

	// StateEstablished indicates the service is fully registered.
	StateEstablished

	// StateConflictDetected indicates a naming conflict was detected during probing.
	// RFC 6762 §8.1: Must choose another name
	StateConflictDetected
)

// String returns the string representation of a State.
func (s State) String() string {
	switch s {
	case StateInitial:
		return "Initial"
	case StateProbing:
		return "Probing"
	case StateAnnouncing:
		return "Announcing"
	case StateEstablished:
		return "Established"
	case StateConflictDetected:
		return "ConflictDetected"
	default:
		return "Unknown"
	}
}
