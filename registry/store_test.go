package registry

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRegistryStoreRegisterAndDiscovery(t *testing.T) {
	store := NewRegistryStore()
	defer func() { _ = store.Close() }()

	store.Register(ServiceInstance{ID: "i1", Name: "svc"})
	store.Register(ServiceInstance{ID: "i2", Name: "svc"})

	instances := store.GetInstances("svc")
	if len(instances) != 2 {
		t.Fatalf("GetInstances() = %d, want 2", len(instances))
	}
	if instances[0].Status != 1 {
		t.Fatalf("Status = %d, want 1 (UP)", instances[0].Status)
	}
	if len(store.GetInstances("missing")) != 0 {
		t.Fatal("GetInstances() on unknown service should be empty")
	}
}

func TestRegistryStoreDeregisterCleansEmptyService(t *testing.T) {
	store := NewRegistryStore()
	defer func() { _ = store.Close() }()

	store.Register(ServiceInstance{ID: "i1", Name: "svc"})
	store.Deregister("svc", "i1")
	if got := store.GetInstances("svc"); len(got) != 0 {
		t.Fatalf("GetInstances() = %v, want empty", got)
	}
	store.mu.RLock()
	_, exists := store.services["svc"]
	store.mu.RUnlock()
	if exists {
		t.Fatal("empty service bucket should be removed")
	}
}

func TestRegistryStoreHeartbeat(t *testing.T) {
	store := NewRegistryStore()
	defer func() { _ = store.Close() }()

	if store.Heartbeat("svc", "i1") {
		t.Fatal("Heartbeat() on unregistered instance should be false")
	}
	store.Register(ServiceInstance{ID: "i1", Name: "svc"})
	if !store.Heartbeat("svc", "i1") {
		t.Fatal("Heartbeat() = false, want true")
	}
}

func TestRegistryStoreEvictsExpiredInstances(t *testing.T) {
	store := NewRegistryStore()
	defer func() { _ = store.Close() }()

	store.Register(ServiceInstance{ID: "stale", Name: "svc"})
	store.Register(ServiceInstance{ID: "fresh", Name: "svc"})

	store.mu.Lock()
	store.services["svc"]["stale"].LastUpdated = time.Now().Unix() - int64((DefaultTTL*evictFactor).Seconds()) - 1
	store.mu.Unlock()

	store.evictExpired(time.Now().Unix())

	instances := store.GetInstances("svc")
	if len(instances) != 1 || instances[0].ID != "fresh" {
		t.Fatalf("after eviction = %v, want only [fresh]", instances)
	}
}

// TestRegistryStoreCloseStopsGoroutine 覆盖"每建一个 store 泄漏一个 goroutine"的缺陷。
func TestRegistryStoreCloseStopsGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()

	stores := make([]*RegistryStore, 0, 20)
	for i := 0; i < 20; i++ {
		stores = append(stores, NewRegistryStore())
	}
	running := runtime.NumGoroutine()
	if running <= before {
		t.Fatalf("expected goroutines to grow, before=%d after=%d", before, running)
	}

	for _, store := range stores {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
	// Close 幂等
	if err := stores[0].Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("goroutines still running after Close: before=%d now=%d", before, runtime.NumGoroutine())
}

// TestRegistryStoreConcurrentAccess 用 -race 暴露 map 并发读写。
func TestRegistryStoreConcurrentAccess(t *testing.T) {
	store := NewRegistryStore()
	defer func() { _ = store.Close() }()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i))
			for j := 0; j < 50; j++ {
				store.Register(ServiceInstance{ID: id, Name: "svc"})
				store.Heartbeat("svc", id)
				store.GetInstances("svc")
				store.Deregister("svc", id)
			}
		}(i)
	}
	wg.Wait()
}
