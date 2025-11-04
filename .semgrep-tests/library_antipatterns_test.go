// Test file for library-specific anti-patterns from research
// Tests rules based on ".archive/research/Designing Premier Go MDNS Library.md"
//
// These are violations found in existing Go mDNS libraries that we must avoid
package test

import (
	"log"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

// ============================================================================
// RESEARCH ANTI-PATTERN 1: Close of Closed Channel
// Source: grandcat/zeroconf issue - "Error: close of closed channel"
// ============================================================================

func BadCloseClosedChannel() {
	ch := make(chan struct{})

	// SHOULD TRIGGER: beacon-close-closed-channel
	close(ch)
	// ... some logic
	close(ch) // PANIC! Channel already closed
}

func BadCloseInMultiplePaths() {
	ch := make(chan string)

	// SHOULD TRIGGER: beacon-close-closed-channel
	if true {
		close(ch)
	}
	// ... more logic
	close(ch) // Could panic if first close executed
}

func GoodCloseWithOnce() {
	ch := make(chan struct{})
	closed := false

	if !closed {
		close(ch)
		closed = true
	}

	// Safe - won't close twice
	if !closed {
		close(ch)
	}
}

func GoodCloseWithDefer() {
	ch := make(chan struct{})
	defer close(ch) // Only closes once when function returns
	// ... work
}

// ============================================================================
// RESEARCH ANTI-PATTERN 2: Global Logger Creation
// Source: "library must *not* create its own logger"
// Mandate: Must accept *slog.Logger via Functional Options
// ============================================================================

// SHOULD TRIGGER: beacon-global-logger-creation
var globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type BadServiceWithOwnLogger struct {
	// SHOULD TRIGGER: beacon-global-logger-creation
	logger *slog.Logger
}

func (s *BadServiceWithOwnLogger) Start() {
	// Library creates its own logger (WRONG!)
	// SHOULD TRIGGER: beacon-global-logger-creation
	s.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	s.logger.Info("service started")
}

// GOOD: Accept logger via option
type GoodServiceWithInjectedLogger struct {
	logger *slog.Logger
}

type Option func(*GoodServiceWithInjectedLogger)

func WithLogger(l *slog.Logger) Option {
	return func(s *GoodServiceWithInjectedLogger) {
		s.logger = l
	}
}

func NewService(opts ...Option) *GoodServiceWithInjectedLogger {
	s := &GoodServiceWithInjectedLogger{
		logger: slog.Default(), // Use default if not provided
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ============================================================================
// RESEARCH ANTI-PATTERN 3: Global Metrics Registration
// Source: "must *not* register metrics globally. This is a 'global state'
//         anti-pattern that pollutes the user's application."
// Mandate: Must accept prometheus.Registerer via Functional Options
// ============================================================================

var (
	// SHOULD TRIGGER: beacon-global-metrics-registration
	packetsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mdns_packets_received_total",
		Help: "Total packets received",
	})
)

func init() {
	// SHOULD TRIGGER: beacon-global-metrics-registration
	prometheus.MustRegister(packetsReceived)
}

type BadMetricsService struct {
	counter prometheus.Counter
}

func (s *BadMetricsService) Start() {
	s.counter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "service_requests_total",
	})

	// Global registration (WRONG!)
	// SHOULD TRIGGER: beacon-global-metrics-registration
	prometheus.MustRegister(s.counter)
}

// GOOD: Accept registerer via option
type GoodMetricsService struct {
	counter prometheus.Counter
	reg     prometheus.Registerer
}

func WithMetricsRegisterer(reg prometheus.Registerer) func(*GoodMetricsService) {
	return func(s *GoodMetricsService) {
		s.reg = reg
	}
}

func NewMetricsService(opts ...func(*GoodMetricsService)) *GoodMetricsService {
	s := &GoodMetricsService{
		reg: prometheus.NewRegistry(), // Default isolated registry
	}
	for _, opt := range opts {
		opt(s)
	}

	// Register to injected registry (CORRECT!)
	s.counter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "service_requests_total",
	})
	s.reg.MustRegister(s.counter)

	return s
}

// ============================================================================
// RESEARCH ANTI-PATTERN 4: Standard log Package Usage
// Source: "Do not use standard log.Printf(). Must use log/slog."
// Mandate: Standardize on Go's built-in structured logging: log/slog
// ============================================================================

func BadStandardLogUsage() {
	// SHOULD TRIGGER: beacon-standard-log-usage
	log.Println("Starting service")

	// SHOULD TRIGGER: beacon-standard-log-usage
	log.Printf("Received packet from %s", "192.168.1.100")

	// SHOULD TRIGGER: beacon-standard-log-usage
	log.Fatal("Service failed to start")
}

func BadStandardLogWithError(err error) {
	// SHOULD TRIGGER: beacon-standard-log-usage
	log.Printf("Error: %v", err)
}

// GOOD: Use slog
func GoodSlogUsage(logger *slog.Logger) {
	logger.Info("Starting service")

	logger.Debug("Received packet",
		"source", "192.168.1.100")

	logger.Error("Service failed to start",
		"error", "connection refused")
}

func GoodSlogWithError(logger *slog.Logger, err error) {
	logger.Error("Operation failed",
		"error", err,
		"component", "transport")
}

// ============================================================================
// Edge Cases and Exceptions
// ============================================================================

// Test files are allowed to use standard log for quick debugging
func TestHelperWithStandardLog() {
	// This should NOT trigger in test files
	log.Println("Debug output in test")
}

// Example code might show both approaches for comparison
func ExampleComparingLoggers() {
	// Documentation example showing migration from log to slog
	// Old way:
	log.Println("message") // OK in examples for demonstration

	// New way:
	slog.Info("message") // Preferred
}
