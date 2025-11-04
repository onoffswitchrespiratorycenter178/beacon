package responder

// Option is a functional option for configuring a Responder.
//
// This pattern allows flexible configuration without breaking API compatibility.
//
// T044: Implement functional options pattern
type Option func(*Responder) error

// WithHostname sets a custom hostname for the responder.
//
// If not provided, the system hostname will be used.
//
// Parameters:
//   - hostname: Custom hostname (e.g., "myhost.local")
//
// Returns:
//   - Option: Configuration function
//
// Example:
//
//	r, err := New(ctx, WithHostname("mydevice.local"))
//
// T044: WithHostname option
func WithHostname(hostname string) Option {
	return func(r *Responder) error {
		r.hostname = hostname
		return nil
	}
}
