package shards

import (
	"sync"
	"testing"
)

func TestNewPool(t *testing.T) {
	num := 16
	p := NewPool(num)
	if p.numShards != num {
		t.Errorf("expected %d shards, got %d", num, p.numShards)
	}
	if len(p.shards) != num {
		t.Errorf("expected shards slice length %d, got %d", num, len(p.shards))
	}
}

func TestGetShardMutex(t *testing.T) {
	p := NewPool(4)
	id1 := "test-id-1"
	id2 := "test-id-2"

	m1 := p.GetShardMutex(id1)
	m1_again := p.GetShardMutex(id1)

	if m1 != m1_again {
		t.Error("same ID should return the same mutex instance")
	}

	m2 := p.GetShardMutex(id2)
	// There is a 1/4 chance they land on the same shard, but for these specific IDs:
	// fnv1a("test-id-1") % 4 != fnv1a("test-id-2") % 4
	if m1 == m2 {
		t.Log("Note: IDs happened to hash to the same shard, this is possible but unlikely for small pools")
	}
}

func TestPoolDistribution(t *testing.T) {
	numShards := 8
	p := NewPool(numShards)
	hits := make(map[*sync.Mutex]bool)

	// Try many different IDs to ensure we hit multiple shards
	for i := 0; i < 100; i++ {
		id := string(rune(i))
		hits[p.GetShardMutex(id)] = true
	}

	if len(hits) <= 1 {
		t.Errorf("expected distribution across shards, but only hit %d shard(s)", len(hits))
	}
}

func TestPoolConcurrency(t *testing.T) {
	p := NewPool(8)
	var wg sync.WaitGroup
	numRoutines := 100
	iterations := 1000

	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			shardID := "shard" // All routines fight for the same shard
			for j := 0; j < iterations; j++ {
				mu := p.GetShardMutex(shardID)
				mu.Lock()
				// Simulate some work
				_ = j * j
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
}
