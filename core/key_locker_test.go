package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewKeyLocker(t *testing.T) {
	kl := NewKeyLocker()
	if kl == nil {
		t.Error("NewKeyLocker() should not return nil")
		return
	}
	if kl.sep != ":" {
		t.Errorf("NewKeyLocker() separator = %v, want :", kl.sep)
	}
}

func TestKeyLocker_Lock_SingleKey(t *testing.T) {
	kl := NewKeyLocker()

	// Test basic lock/unlock
	unlock := kl.Lock("test-key")
	if unlock == nil {
		t.Error("Lock() should return an unlock function")
	}
	unlock()
}

func TestKeyLocker_Lock_MultipleKeys(t *testing.T) {
	kl := NewKeyLocker()

	// Test lock with multiple keys
	unlock := kl.Lock("key1", "key2", 123)
	if unlock == nil {
		t.Error("Lock() should return an unlock function")
	}
	unlock()
}

func TestKeyLocker_Lock_Concurrency(t *testing.T) {
	kl := NewKeyLocker()
	const numGoroutines = 10
	const key = "shared-key"

	var counter int64
	var wg sync.WaitGroup

	// Start multiple goroutines that try to access the same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			unlock := kl.Lock(key)
			defer unlock()

			// Critical section - increment counter
			current := atomic.LoadInt64(&counter)
			time.Sleep(10 * time.Millisecond) // Simulate work
			atomic.StoreInt64(&counter, current+1)
		}()
	}

	wg.Wait()

	if counter != numGoroutines {
		t.Errorf("Counter = %v, want %v. Lock did not provide mutual exclusion", counter, numGoroutines)
	}
}

func TestKeyLocker_Lock_DifferentKeys(t *testing.T) {
	kl := NewKeyLocker()

	// Test that different keys don't block each other
	var wg sync.WaitGroup
	start := make(chan struct{})
	finished := make(chan struct{}, 2)

	// Goroutine 1 locks key1
	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock := kl.Lock("key1")
		defer unlock()

		<-start
		time.Sleep(50 * time.Millisecond)
		finished <- struct{}{}
	}()

	// Goroutine 2 locks key2
	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock := kl.Lock("key2")
		defer unlock()

		<-start
		time.Sleep(50 * time.Millisecond)
		finished <- struct{}{}
	}()

	// Start both goroutines at the same time
	close(start)

	// Both should finish around the same time (not blocking each other)
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for i := 0; i < 2; i++ {
		select {
		case <-finished:
			// Good, goroutine finished
		case <-timer.C:
			t.Error("Goroutines with different keys should not block each other")
			return
		}
	}

	wg.Wait()
}

func TestKeyLocker_Lock_SameKeyBlocking(t *testing.T) {
	kl := NewKeyLocker()
	const key = "blocking-key"

	var order []int
	var mu sync.Mutex

	addToOrder := func(id int) {
		mu.Lock()
		order = append(order, id)
		mu.Unlock()
	}

	var wg sync.WaitGroup

	// First goroutine gets the lock
	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock := kl.Lock(key)
		defer unlock()

		addToOrder(1)
		time.Sleep(100 * time.Millisecond)
		addToOrder(2)
	}()

	// Give first goroutine time to acquire lock
	time.Sleep(10 * time.Millisecond)

	// Second goroutine should be blocked
	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock := kl.Lock(key)
		defer unlock()

		addToOrder(3)
	}()

	wg.Wait()

	// Verify execution order: 1, 2 (from first goroutine), then 3 (from second)
	expected := []int{1, 2, 3}
	mu.Lock()
	if len(order) != len(expected) {
		t.Errorf("Order length = %v, want %v", len(order), len(expected))
		mu.Unlock()
		return
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Order[%d] = %v, want %v. Full order: %v", i, order[i], v, order)
			break
		}
	}
	mu.Unlock()
}

func TestKeyLocker_Lock_KeyGeneration(t *testing.T) {
	kl := NewKeyLocker()

	tests := []struct {
		name string
		keys []any
		want string
	}{
		{
			name: "single string key",
			keys: []any{"test"},
			want: "test",
		},
		{
			name: "multiple string keys",
			keys: []any{"key1", "key2", "key3"},
			want: "key1:key2:key3",
		},
		{
			name: "mixed type keys",
			keys: []any{"user", 123, "resource"},
			want: "user:123:resource",
		},
		{
			name: "numeric keys",
			keys: []any{1, 2, 3},
			want: "1:2:3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the same combination of keys results in the same lock behavior
			unlock1 := kl.Lock(tt.keys...)

			// Start a goroutine to try to acquire the same lock
			blocked := make(chan struct{})
			unblocked := make(chan struct{})

			go func() {
				close(blocked) // Signal that we're about to try to lock
				unlock2 := kl.Lock(tt.keys...)
				close(unblocked) // Signal that we got the lock
				unlock2()
			}()

			// Wait for the goroutine to start attempting to lock
			<-blocked

			// The second lock should be blocked
			select {
			case <-unblocked:
				t.Error("Second lock with same key should be blocked")
			case <-time.After(50 * time.Millisecond):
				// Good, second lock is blocked
			}

			// Now unlock the first lock and second should proceed
			unlock1()

			select {
			case <-unblocked:
				// Good, second lock proceeded after first was unlocked
			case <-time.After(100 * time.Millisecond):
				t.Error("Second lock should proceed after first is unlocked")
			}
		})
	}
}

func TestKeyLocker_Lock_ReferenceCountingCleanup(_ *testing.T) {
	kl := NewKeyLocker()
	const key = "cleanup-test"

	// Acquire and release lock multiple times
	for i := 0; i < 5; i++ {
		unlock := kl.Lock(key)
		unlock()
	}

	// The locks map should clean up unused locks
	// We can't directly access the internal map, but we can verify
	// that multiple locks on the same key still work correctly
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			unlock := kl.Lock(key)
			defer unlock()
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock, reference counting is working
}

func TestKeyLocker_Lock_EmptyKeys(t *testing.T) {
	kl := NewKeyLocker()

	// Test with no keys (should still work)
	unlock := kl.Lock()
	if unlock == nil {
		t.Error("Lock() with no keys should still return unlock function")
	}
	unlock()

	// Test with empty string keys
	unlock2 := kl.Lock("", "", "")
	if unlock2 == nil {
		t.Error("Lock() with empty string keys should still return unlock function")
	}
	unlock2()
}

func TestKeyLocker_Lock_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	kl := NewKeyLocker()
	const numGoroutines = 20
	const numKeys = 5

	var wg sync.WaitGroup
	var results [numKeys]int64

	// Start many goroutines accessing different keys
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			keyIndex := id % numKeys
			unlock := kl.Lock("key", keyIndex)
			defer unlock()

			// Increment counter for this key atomically
			atomic.AddInt64(&results[keyIndex], 1)
			time.Sleep(1 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	// Verify all operations completed
	var total int64
	for _, count := range results {
		total += count
	}

	if total != numGoroutines {
		t.Errorf("Total operations = %v, want %v", total, numGoroutines)
	}
}
