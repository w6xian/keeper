package registry

type ServiceInstance struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Weight      int               `json:"weight"`
	Status      int               `json:"status"` // 1: UP, 0: DOWN
	LastUpdated int64             `json:"last_updated"`
}

type RegisterRequest struct {
	Instance ServiceInstance `json:"instance"`
}

type RegisterResponse struct {
	TTL int64 `json:"ttl"` // Seconds
}

type DeregisterRequest struct {
	ServiceName string `json:"service_name"`
	InstanceID  string `json:"instance_id"`
}

type HeartbeatRequest struct {
	ServiceName string `json:"service_name"`
	InstanceID  string `json:"instance_id"`
}

type DiscoveryRequest struct {
	ServiceName string   `json:"service_name"`
	Tags        []string `json:"tags,omitempty"`
}

type DiscoveryResponse struct {
	Instances []ServiceInstance `json:"instances"`
}
