package di

import (
	"sync"
	"testing"
	"time"

	"github.com/kyanite/focus/internal/engine"
	"github.com/kyanite/focus/internal/repository"
)

// deadlockGuard bounds each wait below. Before the fix, GetEngine re-entered
// c.mu and blocked forever; this guard turns that hang into a test failure
// instead of a stuck test binary.
const deadlockGuard = 5 * time.Second

// TestGetEngineReturns is the direct regression for the DI self-deadlock:
// GetEngine used to call GetRepository while holding c.mu.
func TestGetEngineReturns(t *testing.T) {
	c := NewContainerWithPath(t.TempDir() + "/tasks.json")

	done := make(chan *engine.Engine, 1)
	go func() { done <- c.GetEngine() }()

	select {
	case eng := <-done:
		if eng == nil {
			t.Fatal("GetEngine returned nil")
		}
	case <-time.After(deadlockGuard):
		t.Fatal("GetEngine did not return within 5s: the container deadlocked on its own mutex")
	}
}

// TestConcurrentGetEngineAndGetRepository calls both accessors from several
// goroutines at once and requires all of them to finish. A re-entrant lock
// deadlock never finishes, so this test fails rather than hangs.
func TestConcurrentGetEngineAndGetRepository(t *testing.T) {
	c := NewContainerWithPath(t.TempDir() + "/tasks.json")
	const workers = 8

	start := make(chan struct{})
	engines := make([]*engine.Engine, workers)
	repos := make([]repository.Repository, workers)

	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				repos[i] = c.GetRepository()
				engines[i] = c.GetEngine()
			} else {
				engines[i] = c.GetEngine()
				repos[i] = c.GetRepository()
			}
		}(i)
	}

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	close(start)

	select {
	case <-finished:
	case <-time.After(deadlockGuard):
		t.Fatal("concurrent GetEngine/GetRepository calls did not finish within 5s: mutex re-entry")
	}

	for i := 1; i < workers; i++ {
		if engines[i] != engines[0] {
			t.Errorf("GetEngine returned different instances (worker %d)", i)
		}
		if repos[i] != repos[0] {
			t.Errorf("GetRepository returned different instances (worker %d)", i)
		}
	}
}
