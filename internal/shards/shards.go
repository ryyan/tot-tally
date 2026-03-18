// shards.go provides a thread-safe sharded mutex pool for high concurrency file access.
package shards

import (
	"hash/fnv"
	"sync"
)

// paddedMutex wraps a standard sync.Mutex with extra padding to ensure it occupies
// its own CPU cache line. On most modern CPUs, a cache line is 64 bytes.
//
// Why this is used:
// If multiple mutexes share the same cache line, a CPU core updating one mutex
// will invalidate the cache for all other mutexes on that same line (False Sharing).
// This causes significant performance degradation in high-concurrency scenarios
// as different cores fight for ownership of the same memory segment.
//
// By padding the struct to 64 bytes, we ensure that each mutex in our pool
// can be locked and unlocked by different cores simultaneously without any
// hardware-level interference.
type paddedMutex struct {
	sync.Mutex
	_ [64 - 8]byte // Pad to 64 bytes (assuming 8-byte Mutex on 64-bit systems)
}

// Pool implements a sharded mutex system.
// Instead of a single global lock or a map that grows indefinitely, a sharded
// pool provides a fixed-size set of mutexes. This guarantees high concurrency
// with a strictly predictable and capped memory footprint.
type Pool struct {
	shards    []paddedMutex
	numShards int
}

// NewPool initializes a new sharded mutex pool with the specified number of shards.
// The number of shards should ideally be a power of 2 for optimal distribution.
func NewPool(num int) *Pool {
	return &Pool{
		shards:    make([]paddedMutex, num),
		numShards: num,
	}
}

// GetShardMutex selects a mutex from the pool by hashing the input string.
// It uses FNV-1a hashing for its excellent distribution and low CPU overhead.
// This allows independent Tot records or IP limits to be updated in parallel
// while protecting individual records from race conditions.
func (p *Pool) GetShardMutex(id string) *sync.Mutex {
	h := fnv.New32a()
	h.Write([]byte(id))
	// Use uint32 for the modulo to ensure positive index mapping.
	return &p.shards[h.Sum32()%uint32(p.numShards)].Mutex
}
