package registry

import (
	"log"
	"sync"
	"time"
)

const (
	// DefaultTTL 实例心跳的有效期。
	DefaultTTL = 10 * time.Second
	// evictFactor 超过 TTL 多少倍才剔除：给心跳抖动/网络延迟留余量。
	evictFactor = 3
	// evictInterval 过期扫描间隔。
	evictInterval = 5 * time.Second
)

type RegistryStore struct {
	services map[string]map[string]*ServiceInstance // map[ServiceName]map[InstanceID]*Instance
	mu       sync.RWMutex

	closeOnce sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewRegistryStore() *RegistryStore {
	store := &RegistryStore{
		services: make(map[string]map[string]*ServiceInstance),
		stopCh:   make(chan struct{}),
	}
	store.wg.Add(1)
	go store.evictionLoop()
	return store
}

// Close 停止后台剔除协程。
//
// 旧实现里 evictionLoop 是一个永不退出的 goroutine + 永不 Stop 的 ticker：
// 每建一个 store 就泄漏一个，长跑进程里会一直累积。
// Close 幂等，可重复调用。
func (s *RegistryStore) Close() error {
	s.closeOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
	return nil
}

func (s *RegistryStore) Register(instance ServiceInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.services[instance.Name]; !ok {
		s.services[instance.Name] = make(map[string]*ServiceInstance)
	}
	instance.LastUpdated = time.Now().Unix()
	instance.Status = 1 // UP
	s.services[instance.Name][instance.ID] = &instance
	log.Printf("Service registered %s %s", instance.Name, instance.ID)
}

func (s *RegistryStore) Deregister(serviceName, instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if instances, ok := s.services[serviceName]; ok {
		delete(instances, instanceID)
		if len(instances) == 0 {
			delete(s.services, serviceName)
		}
		log.Printf("Service deregistered name=%s id=%s", serviceName, instanceID)
	}
}

func (s *RegistryStore) Heartbeat(serviceName, instanceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if instances, ok := s.services[serviceName]; ok {
		if instance, ok := instances[instanceID]; ok {
			instance.LastUpdated = time.Now().Unix()
			log.Printf("Heartbeat received %s %s", serviceName, instanceID)
			return true
		}
	}
	return false
}

func (s *RegistryStore) GetInstances(serviceName string) []ServiceInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ServiceInstance
	if instances, ok := s.services[serviceName]; ok {
		for _, instance := range instances {
			if instance.Status == 1 {
				result = append(result, *instance)
			}
		}
	}
	return result
}

func (s *RegistryStore) evictionLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(evictInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evictExpired(time.Now().Unix())
		}
	}
}

// evictExpired 剔除超过 evictFactor*TTL 没有心跳的实例。
func (s *RegistryStore) evictExpired(now int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expireAfter := int64((DefaultTTL * evictFactor).Seconds())
	for serviceName, instances := range s.services {
		for id, instance := range instances {
			if now-instance.LastUpdated > expireAfter {
				log.Printf("Evicting expired instance %s %s", serviceName, id)
				delete(instances, id)
			}
		}
		if len(instances) == 0 {
			delete(s.services, serviceName)
		}
	}
}
