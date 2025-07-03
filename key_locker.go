package vast_client

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

type refLock struct {
	mu  sync.Mutex
	ref int32
}

type KeyLocker struct {
	locks sync.Map
	sep   string
}

// NewKeyLocker creates a new KeyLocker.
func NewKeyLocker() *KeyLocker {
	return &KeyLocker{sep: ":"}
}

// Lock returns a function that will unlock the key when called
func (kl *KeyLocker) Lock(keys ...any) func() {
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%v", k))
	}
	combinedKey := strings.Join(parts, kl.sep)

	lockIface, _ := kl.locks.LoadOrStore(combinedKey, &refLock{})
	lock := lockIface.(*refLock)

	atomic.AddInt32(&lock.ref, 1)
	lock.mu.Lock()

	// Return a closure that unlocks and cleans up
	return func() {
		lock.mu.Unlock()
		if atomic.AddInt32(&lock.ref, -1) == 0 {
			kl.locks.Delete(combinedKey)
		}
	}
}
