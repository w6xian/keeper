package registry

import (
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultTTL = 10 * time.Second
)

type RegistryStore struct {
	services map[string]map[string]*ServiceInstance // map[ServiceName]map[InstanceID]*Instance
	mu       sync.RWMutex
}

func NewRegistryStore() *RegistryStore {
	store := &RegistryStore{
		services: make(map[string]map[string]*ServiceInstance),
	}
	// Start eviction routine
	go store.evictionLoop()
	return store
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
		log.Printf("Service deregistered", zap.String("name", serviceName), zap.String("id", instanceID))
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
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now().Unix()
		for serviceName, instances := range s.services {
			for id, instance := range instances {
				if now-instance.LastUpdated > int64(DefaultTTL.Seconds()*3) {
					log.Printf("Evicting expired instance %s %s", serviceName, id)
					delete(instances, id)
				}
			}
			if len(instances) == 0 {
				delete(s.services, serviceName)
			}
		}
		s.mu.Unlock()
	}
}
