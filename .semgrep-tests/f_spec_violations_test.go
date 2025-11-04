// F-Spec MUST/SHOULD Enforcement Tests
// TDD: RED → GREEN → REFACTOR
//
// This file contains intentional violations of F-Spec requirements
// to validate Semgrep rules catch them.

package test

import (
	"context"
	"fmt"
	"net"
	"time"
)

// ==============================================================================
// F-3: Error Handling - Error Wrapping
// ==============================================================================

// BadErrorWrapPercentV wraps error with %v instead of %w
func BadErrorWrapPercentV() error {
	err := fmt.Errorf("something failed")

	// ruleid: beacon-error-wrap-percent-v
	// ❌ Uses %v - breaks error chain
	return fmt.Errorf("context: %v", err)
}

// GoodErrorWrapPercentW wraps error correctly with %w
func GoodErrorWrapPercentW() error {
	err := fmt.Errorf("something failed")

	// ok: Uses %w - preserves error chain
	return fmt.Errorf("context: %w", err)
}

// ==============================================================================
// F-4: Concurrency - RFC Timing Constants Must Be in Protocol Package
// ==============================================================================

// BadProbeWithLocalTimingConst uses local const for RFC-mandated timing
func BadProbeWithLocalTimingConst(ctx context.Context) {
	// ruleid: beacon-rfc-timing-local-const
	// ❌ RFC 6762 §8.1 mandates 250ms - should be protocol.ProbeInterval
	const probeInterval = 250 * time.Millisecond

	time.Sleep(probeInterval)
}

// BadAnnounceWithLocalTimingConst uses local const for RFC-mandated timing
func BadAnnounceWithLocalTimingConst(ctx context.Context) {
	// ruleid: beacon-rfc-timing-local-const
	// ❌ RFC 6762 §8.3 mandates intervals - should be protocol constant
	const announceInterval = 1 * time.Second

	time.Sleep(announceInterval)
}

// GoodProbeWithProtocolConst uses protocol package constant
func GoodProbeWithProtocolConst(ctx context.Context) {
	// ok: Uses protocol package constant (would be defined there)
	// const from protocol package (not local)
	interval := 250 * time.Millisecond // OK if this comes from protocol.ProbeInterval
	time.Sleep(interval)
}

// ==============================================================================
// F-9: Transport - net.ListenMulticastUDP() Forbidden
// ==============================================================================

// BadListenMulticastUDP uses forbidden function
func BadListenMulticastUDP() (net.PacketConn, error) {
	addr, _ := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")

	// ruleid: beacon-listen-multicast-udp
	// ❌ F-9 MUST NOT use net.ListenMulticastUDP (can't set socket options)
	return net.ListenMulticastUDP("udp4", nil, addr)
}

// GoodListenWithSocketOptions uses platform-specific socket creation
func GoodListenWithSocketOptions() (net.PacketConn, error) {
	// ok: Would use platform-specific socket creation with SO_REUSEPORT
	// This is a placeholder - actual impl would be in internal/network/socket_*.go
	return nil, nil
}

// ==============================================================================
// F-6: Logging - TXT Record Values MUST NOT Be Logged
// ==============================================================================

type TXTRecord struct {
	Keys   []string
	Values []string
	Data   []byte
}

// BadLogTXTRecordValues logs TXT record values (may contain secrets)
func BadLogTXTRecordValues(record TXTRecord, logger Logger) {
	// ruleid: beacon-txt-record-value-logging
	// ❌ TXT values may contain API keys, passwords, tokens
	logger.Debug("TXT record", "values", record.Values)

	// ruleid: beacon-txt-record-value-logging
	// ❌ Raw data may contain secrets
	logger.Info("TXT data", "data", record.Data)
}

// GoodLogTXTRecordKeys logs only TXT record keys (safe)
func GoodLogTXTRecordKeys(record TXTRecord, logger Logger) {
	// ok: Only logs keys, not values
	logger.Debug("TXT record", "keys", record.Keys)
}

// ==============================================================================
// F-4: Concurrency - Goroutine Without Context Check
// ==============================================================================

// BadGoroutineWithoutContextCheck launches goroutine that never exits
func BadGoroutineWithoutContextCheck(ctx context.Context) {
	// ruleid: beacon-goroutine-no-context
	// ❌ Goroutine leaks - no context check in loop
	go func() {
		for {
			doWork()
			time.Sleep(1 * time.Second)
		}
	}()
}

// GoodGoroutineWithContextCheck respects context cancellation
func GoodGoroutineWithContextCheck(ctx context.Context) {
	// ok: Checks context.Done() in loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				doWork()
			}
		}
	}()
}

// ==============================================================================
// F-5: Configuration - RFC MUST Requirements Made Configurable
// ==============================================================================

type Config struct {
	ProbeCount int
}

type Option func(*Config)

// BadProbeCountConfigurable makes RFC MUST requirement configurable
func BadProbeCountConfigurable() {
	// ruleid: beacon-rfc-must-configurable
	// ❌ RFC 6762 §8.1 mandates exactly 3 probes - cannot be configurable
	WithProbeCount := func(count int) Option {
		return func(c *Config) {
			c.ProbeCount = count
		}
	}
	_ = WithProbeCount
}

// GoodProbeCountConstant uses constant for RFC requirement
func GoodProbeCountConstant() {
	// ok: Probe count is a constant, not configurable
	const ProbeCount = 3 // RFC 6762 §8.1
	_ = ProbeCount
}

// ==============================================================================
// Helper Types
// ==============================================================================

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
}

func doWork() {
	// Placeholder
}
