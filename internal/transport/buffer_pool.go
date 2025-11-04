package transport

import (
	"sync"
)

// Buffer pool for receive operations per research.md Topic 2 (FR-003)
//
// Problem: UDPv4Transport.Receive() allocates 9KB buffer on each call
// Solution: sync.Pool for buffer reuse
// Expected Impact: 900KB/sec → near-zero allocations (≥80% reduction)
//
// Pattern from research.md Topic 2:
//   bufPtr := GetBuffer()
//   defer PutBuffer(bufPtr)
//   buf := *bufPtr
//   ... use buffer ...

// bufferPool is a sync.Pool for 9000-byte receive buffers.
//
// sync.Pool provides:
// - Thread-safe buffer reuse
// - Automatic GC cleanup of unused buffers
// - Zero allocation on hot path (after warmup)
//
// T050: Minimal pool to make T044-T046 pass
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Allocate 9KB buffer for mDNS packets
		// RFC 6762 §17: mDNS messages can exceed 512 bytes (jumbo frames up to 9000)
		buf := make([]byte, 9000)
		return &buf
	},
}

// GetBuffer returns a pointer to a 9000-byte buffer from the pool.
//
// Caller MUST call PutBuffer() to return the buffer (use defer).
//
// Returns:
//   - *[]byte: Pointer to 9KB buffer
//
// T051: Export GetBuffer() function
func GetBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// PutBuffer returns a buffer to the pool for reuse.
//
// Caller MUST NOT use the buffer after calling PutBuffer().
// Best practice: Use defer PutBuffer(bufPtr) immediately after GetBuffer().
//
// Parameters:
//   - bufPtr: Pointer to buffer (from GetBuffer())
//
// T052: Export PutBuffer() function
func PutBuffer(bufPtr *[]byte) {
	// Clear buffer before returning to pool (security: no data leakage)
	// Note: This adds overhead, but prevents accidental data leakage between receives
	buf := *bufPtr
	for i := range buf {
		buf[i] = 0
	}

	bufferPool.Put(bufPtr)
}
