package config

import (
	"sync"
	"testing"
	"time"
)

// resetReloadHooks clears registered hooks so tests don't interfere with each other.
func resetReloadHooks() {
	reloadHooksMutex.Lock()
	reloadHooks = nil
	reloadHooksMutex.Unlock()
}

func TestRunReloadHooksInvokesAll(t *testing.T) {
	resetReloadHooks()
	defer resetReloadHooks()

	var calls int
	OnReload(func() { calls++ })
	OnReload(func() { calls++ })

	runReloadHooks()

	if calls != 2 {
		t.Errorf("expected 2 hook calls, got %d", calls)
	}
}

func TestRunReloadHooksNoHooks(t *testing.T) {
	resetReloadHooks()
	defer resetReloadHooks()

	// Should not panic with no registered hooks.
	runReloadHooks()
}

// TestRunReloadHooksHookMayCallGet guards against a regression where hooks were
// invoked while the config write lock was held, deadlocking any hook that reads
// the config via Get() (which acquires the read lock).
func TestRunReloadHooksHookMayCallGet(t *testing.T) {
	resetReloadHooks()
	defer resetReloadHooks()

	done := make(chan struct{})
	OnReload(func() {
		_ = Get() // would deadlock if a write lock were held during the callback
		close(done)
	})

	go runReloadHooks()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("hook calling Get() deadlocked")
	}
}

// TestOnReloadConcurrentRegistration ensures hook registration is safe under
// concurrent use.
func TestOnReloadConcurrentRegistration(t *testing.T) {
	resetReloadHooks()
	defer resetReloadHooks()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			OnReload(func() {})
		}()
	}
	wg.Wait()

	reloadHooksMutex.Lock()
	n := len(reloadHooks)
	reloadHooksMutex.Unlock()

	if n != 50 {
		t.Errorf("expected 50 registered hooks, got %d", n)
	}
}
