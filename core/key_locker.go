package core

import (
	"fmt"
	"strings"
	"sync"
)

type refLock struct {
	mu  sync.Mutex
	ref int // guarded by KeyLocker.mu
}

type KeyLocker struct {
	mu    sync.Mutex // guards locks and every refLock.ref
	locks map[string]*refLock
	sep   string
}

// NewKeyLocker creates a new KeyLocker.
func NewKeyLocker() *KeyLocker {
	return &KeyLocker{sep: ":", locks: make(map[string]*refLock)}
}

// Lock returns a function that will unlock the key when called
func (kl *KeyLocker) Lock(keys ...any) func() {
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%v", k))
	}
	combinedKey := strings.Join(parts, kl.sep)

	// Look up (or create) the per-key lock and take a reference in one
	// critical section. Doing these in two steps lets a concurrent unlock
	// delete the entry in between, so a later caller creates a second lock
	// for the same key and two holders run at the same time.
	kl.mu.Lock()
	if kl.locks == nil {
		kl.locks = make(map[string]*refLock)
	}
	lock, ok := kl.locks[combinedKey]
	if !ok {
		lock = &refLock{}
		kl.locks[combinedKey] = lock
	}
	lock.ref++
	kl.mu.Unlock()

	lock.mu.Lock()

	// Return a closure that unlocks and cleans up
	return func() {
		lock.mu.Unlock()
		kl.mu.Lock()
		lock.ref--
		if lock.ref == 0 {
			delete(kl.locks, combinedKey)
		}
		kl.mu.Unlock()
	}
}
