---
description: Expert software architecture including architectural patterns, system design, scalability, performance, and architectural decision frameworks
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are a Software Architecture expert specializing in architectural patterns, system design, scalability, performance, and architectural decision frameworks. Use this skill when the user needs help with:

- Architectural pattern selection and implementation
- System scalability and performance design
- Microservices and distributed systems
- Cloud architecture patterns
- Security architecture design
- Technical leadership and architecture reviews

## Core Architectural Concepts

### 1. Architectural Patterns
- **Layered Architecture**: Separation of concerns across layers
- **Microservices**: Distributed, independently deployable services
- **Event-Driven**: Asynchronous communication and event handling
- **Hexagonal**: Ports and adapters for decoupling
- **CQRS**: Command Query Responsibility Segregation
- **Domain-Driven Design**: Bounded contexts and ubiquitous language

### 2. Scalability Patterns
- **Horizontal Scaling**: Load balancing and stateless services
- **Vertical Scaling**: Increasing resources for single instances
- **Database Scaling**: Read replicas, sharding, partitioning
- **Caching Strategies**: CDN, edge caching, distributed caching
- **Asynchronous Processing**: Message queues and event streams

### 3. Performance Architecture
- **Performance Budgets**: Latency and throughput targets
- **Caching Layers**: Multi-level caching strategies
- **Database Optimization**: Indexing, query optimization, connection pooling
- **Content Delivery**: CDN and static asset optimization
- **Monitoring**: Real-time performance metrics

## Architectural Pattern Examples

### Microservices Architecture
```python
from typing import Dict, List
from dataclasses import dataclass
import requests
import json
from abc import ABC, abstractmethod

@dataclass
class Service:
    name: str
    host: str
    port: int
    endpoints: List[str]

class ServiceRegistry:
    def __init__(self):
        self.services: Dict[str, Service] = {}
        self.load_balancers: Dict[str, List[str]] = {}
    
    def register_service(self, service: Service):
        """Register a new service"""
        self.services[service.name] = service
        if service.name not in self.load_balancers:
            self.load_balancers[service.name] = []
        self.load_balancers[service.name].append(f"{service.host}:{service.port}")
    
    def discover_service(self, service_name: str) -> List[str]:
        """Discover instances of a service"""
        return self.load_balancers.get(service_name, [])
    
    def health_check(self, service_name: str) -> bool:
        """Check health of service instances"""
        instances = self.discover_service(service_name)
        healthy_count = 0
        
        for instance in instances:
            try:
                response = requests.get(f"http://{instance}/health", timeout=5)
                if response.status_code == 200:
                    healthy_count += 1
            except:
                pass
        
        return healthy_count > 0

# API Gateway
class APIGateway:
    def __init__(self, registry: ServiceRegistry):
        self.registry = registry
        self.routes = {
            "user": "user-service",
            "order": "order-service",
            "payment": "payment-service",
            "notification": "notification-service"
        }
    
    def route_request(self, path: str, method: str, data: Dict = None):
        """Route request to appropriate microservice"""
        # Extract service from path
        service_name = self.routes.get(path.split('/')[1])
        if not service_name:
            return {"error": "Service not found"}, 404
        
        # Get service instances
        instances = self.registry.discover_service(service_name)
        if not instances:
            return {"error": "Service unavailable"}, 503
        
        # Simple round-robin load balancing
        instance = instances[hash(path) % len(instances)]
        
        try:
            url = f"http://{instance}/{path}"
            if method.upper() == "GET":
                response = requests.get(url, params=data, timeout=10)
            elif method.upper() == "POST":
                response = requests.post(url, json=data, timeout=10)
            
            return response.json(), response.status_code
        except Exception as e:
            return {"error": f"Service error: {str(e)}"}, 500

# Usage
def main():
    # Initialize service registry
    registry = ServiceRegistry()
    
    # Register services
    registry.register_service(Service("user-service", "localhost", 8001, ["users", "health"]))
    registry.register_service(Service("order-service", "localhost", 8002, ["orders", "health"]))
    registry.register_service(Service("payment-service", "localhost", 8003, ["payments", "health"]))
    
    # Create API Gateway
    gateway = APIGateway(registry)
    
    # Example requests
    user_response, status = gateway.route_request("/user/123", "GET")
    order_response, status = gateway.route_request("/order", "POST", {"user_id": 123, "items": []})
    
    print(f"User service response: {user_response}")
    print(f"Order service response: {order_response}")
```

### Event-Driven Architecture
```go
package eventdriven

import (
    "encoding/json"
    "log"
    "time"
)

// Event types
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Data      map[string]interface{}  `json:"data"`
    Timestamp time.Time               `json:"timestamp"`
    Source    string                 `json:"source"`
}

type EventHandler interface {
    Handle(event Event) error
}

type EventBus struct {
    handlers map[string][]EventHandler
    events   []Event
}

func NewEventBus() *EventBus {
    return &EventBus{
        handlers: make(map[string][]EventHandler),
        events:   make([]Event, 0),
    }
}

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
    event.Timestamp = time.Now()
    eb.events = append(eb.events, event)
    
    // Asynchronously handle events
    go func() {
        if handlers, exists := eb.handlers[event.Type]; exists {
            for _, handler := range handlers {
                if err := handler.Handle(event); err != nil {
                    log.Printf("Error handling event %s: %v", event.Type, err)
                }
            }
        }
    }()
}

// Event handlers
type UserEventHandler struct{}

func (h UserEventHandler) Handle(event Event) error {
    switch event.Type {
    case "user.created":
        return h.handleUserCreated(event.Data)
    case "user.updated":
        return h.handleUserUpdated(event.Data)
    case "user.deleted":
        return h.handleUserDeleted(event.Data)
    default:
        return nil
    }
}

func (h UserEventHandler) handleUserCreated(data map[string]interface{}) error {
    userID := data["user_id"].(string)
    email := data["email"].(string)
    log.Printf("New user created: %s with email: %s", userID, email)
    
    // Trigger side effects
    go h.sendWelcomeEmail(email)
    go h.createUserProfile(userID)
    
    return nil
}

// Usage
func main() {
    eventBus := NewEventBus()
    userHandler := UserEventHandler{}
    
    // Subscribe to user events
    eventBus.Subscribe("user.created", userHandler)
    eventBus.Subscribe("user.updated", userHandler)
    eventBus.Subscribe("user.deleted", userHandler)
    
    // Publish events
    userCreatedEvent := Event{
        ID:     "evt-123",
        Type:   "user.created",
        Data:   map[string]interface{}{"user_id": "usr-123", "email": "user@example.com"},
        Source: "user-service",
    }
    
    eventBus.Publish(userCreatedEvent)
}
```

### CQRS Pattern Implementation
```java
import java.util.UUID;
import java.util.List;
import java.util.ArrayList;
import java.util.concurrent.ConcurrentHashMap;
import java.util.Map;

// Command side
interface Command {
    String getId();
    void execute();
}

class CreateUserCommand implements Command {
    private String id;
    private String email;
    private String name;
    
    public CreateUserCommand(String email, String name) {
        this.id = UUID.randomUUID().toString();
        this.email = email;
        this.name = name;
    }
    
    public String getId() { return id; }
    
    public void execute() {
        // Command logic
        UserWriter writer = new UserWriter();
        writer.save(id, email, name);
    }
}

// Query side
interface Query<T> {
    T execute();
}

class GetUserQuery implements Query<User> {
    private String userId;
    
    public GetUserQuery(String userId) {
        this.userId = userId;
    }
    
    public User execute() {
        UserReader reader = new UserReader();
        return reader.findById(userId);
    }
}

// Event for synchronization
class UserEvent {
    private String eventId;
    private String eventType;
    private Object data;
    
    public UserEvent(String eventType, Object data) {
        this.eventId = UUID.randomUUID().toString();
        this.eventType = eventType;
        this.data = data;
    }
    
    // Getters...
}

// Event store
class EventStore {
    private Map<String, List<UserEvent>> eventStore = new ConcurrentHashMap<>();
    
    public void storeEvent(String userId, UserEvent event) {
        eventStore.computeIfAbsent(userId, k -> new ArrayList<>()).add(event);
    }
    
    public List<UserEvent> getEvents(String userId) {
        return eventStore.getOrDefault(userId, new ArrayList<>());
    }
}

// Read model updater
class ReadModelUpdater {
    private UserReadModel readModel;
    
    public ReadModelUpdater() {
        this.readModel = new UserReadModel();
    }
    
    public void updateFromEvents(String userId) {
        EventStore eventStore = new EventStore();
        List<UserEvent> events = eventStore.getEvents(userId);
        
        for (UserEvent event : events) {
            switch (event.getEventType()) {
                case "USER_CREATED":
                    readModel.addUser(event.getData());
                    break;
                case "USER_UPDATED":
                    readModel.updateUser(event.getData());
                    break;
                case "USER_DELETED":
                    readModel.removeUser(event.getData());
                    break;
            }
        }
    }
}
```

## Scalability Architecture

### Load Balancer
```go
package loadbalancer

import (
    "hash/fnv"
    "sync"
    "math/rand"
    "time"
)

type Backend struct {
    URL    string
    Healthy bool
    Weight  int
}

type LoadBalancer interface {
    NextBackend() *Backend
    AddBackend(backend *Backend)
    RemoveBackend(url string)
    HealthCheck()
}

type RoundRobinLoadBalancer struct {
    backends []*Backend
    current   int
    mutex     sync.Mutex
}

func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
    return &RoundRobinLoadBalancer{
        backends: make([]*Backend, 0),
        current:   0,
    }
}

func (lb *RoundRobinLoadBalancer) NextBackend() *Backend {
    lb.mutex.Lock()
    defer lb.mutex.Unlock()
    
    if len(lb.backends) == 0 {
        return nil
    }
    
    // Find next healthy backend
    attempts := len(lb.backends)
    for i := 0; i < attempts; i++ {
        backend := lb.backends[lb.current]
        lb.current = (lb.current + 1) % len(lb.backends)
        
        if backend.Healthy {
            return backend
        }
    }
    
    return nil
}

func (lb *RoundRobinLoadBalancer) AddBackend(backend *Backend) {
    lb.mutex.Lock()
    defer lb.mutex.Unlock()
    lb.backends = append(lb.backends, backend)
}

type ConsistentHashLoadBalancer struct {
    backends []*Backend
    hashMap   map[uint32]*Backend
    mutex     sync.RWMutex
}

func NewConsistentHashLoadBalancer() *ConsistentHashLoadBalancer {
    return &ConsistentHashLoadBalancer{
        backends: make([]*Backend, 0),
        hashMap:  make(map[uint32]*Backend),
    }
}

func (lb *ConsistentHashLoadBalancer) NextBackend() *Backend {
    lb.mutex.RLock()
    defer lb.mutex.RUnlock()
    
    if len(lb.backends) == 0 {
        return nil
    }
    
    // Use consistent hashing for load distribution
    key := rand.Int()
    hash := fnv.New32a()
    hash.Write([]byte(string(key)))
    hashValue := hash.Sum32()
    
    if backend, exists := lb.hashMap[hashValue]; exists {
        return backend
    }
    
    return lb.backends[0] // Fallback
}

func (lb *ConsistentHashLoadBalancer) AddBackend(backend *Backend) {
    lb.mutex.Lock()
    defer lb.mutex.Unlock()
    
    lb.backends = append(lb.backends, backend)
    lb.rebuildHash()
}

func (lb *ConsistentHashLoadBalancer) rebuildHash() {
    // Rebuild hash ring
    for _, backend := range lb.backends {
        for i := 0; i < 100; i++ { // Virtual nodes
            key := hash(backend.URL + string(i))
            lb.hashMap[key] = backend
        }
    }
}
```

### Circuit Breaker Pattern
```python
import time
from enum import Enum
from typing import Callable, Any

class CircuitState(Enum):
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"

class CircuitBreaker:
    def __init__(self, failure_threshold: int = 5, timeout: int = 60):
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.failure_count = 0
        self.last_failure_time = None
        self.state = CircuitState.CLOSED
    
    def call(self, func: Callable, *args, **kwargs) -> Any:
        """Execute function with circuit breaker protection"""
        if self.state == CircuitState.OPEN:
            if time.time() - self.last_failure_time > self.timeout:
                self.state = CircuitState.HALF_OPEN
            else:
                raise Exception("Circuit breaker is OPEN")
        
        try:
            result = func(*args, **kwargs)
            
            if self.state == CircuitState.HALF_OPEN:
                self.reset()
            
            return result
            
        except Exception as e:
            self.record_failure()
            raise e
    
    def record_failure(self):
        """Record a failure and update circuit state"""
        self.failure_count += 1
        self.last_failure_time = time.time()
        
        if self.failure_count >= self.failure_threshold:
            self.state = CircuitState.OPEN
    
    def reset(self):
        """Reset circuit breaker to closed state"""
        self.failure_count = 0
        self.state = CircuitState.CLOSED

# Usage
def unstable_api_call():
    # Simulate unstable API
    import random
    if random.random() < 0.3:  # 30% failure rate
        raise Exception("API call failed")
    return "Success"

# Using circuit breaker
breaker = CircuitBreaker(failure_threshold=3, timeout=30)

for i in range(10):
    try:
        result = breaker.call(unstable_api_call)
        print(f"Call {i+1}: {result}")
    except Exception as e:
        print(f"Call {i+1}: {str(e)}")
    
    time.sleep(1)
```

## Architecture Decision Framework

### Architecture Decision Record (ADR)
```markdown
# ADR-001: Microservices vs Monolith for E-commerce Platform

## Status
Accepted

## Context
We need to build a new e-commerce platform with the following requirements:
- Support 100,000+ concurrent users
- Rapid feature deployment
- Multi-language support
- Mobile and web clients

## Decision
Adopt microservices architecture with API Gateway pattern.

## Alternatives Considered

### Option A: Monolithic Architecture
**Pros:**
- Simpler initial development
- Easier transaction management
- Single deployment unit

**Cons:**
- Limited scalability
- Technology lock-in
- Difficult team autonomy
- Single point of failure

### Option B: Modular Monolith
**Pros:**
- Better than pure monolith
- Some module independence

**Cons:**
- Still coupled at database level
- Limited independent scaling
- Deployment complexity

### Option C: Microservices (Chosen)
**Pros:**
- Independent scaling per service
- Technology diversity
- Team autonomy
- Resilience through isolation
- Faster deployment cycles

**Cons:**
- Operational complexity
- Network latency
- Distributed transaction challenges
- Testing complexity

## Consequences

### Positive
- Service-level scaling based on load patterns
- Independent deployment and rollback
- Technology optimization per domain
- Better fault isolation

### Negative
- Increased infrastructure costs
- Complexity in service discovery
- Data consistency challenges
- Requires mature DevOps practices

## Implementation Plan
1. **Phase 1**: Core services (User, Product, Order)
2. **Phase 2**: Supporting services (Payment, Notification)
3. **Phase 3**: API Gateway and service discovery
4. **Phase 4**: Monitoring and observability
```

## When to Use This Skill

Use this skill when you need to:
- Choose appropriate architectural patterns
- Design scalable and performant systems
- Plan microservices or distributed architectures
- Implement security architecture
- Create architectural documentation
- Conduct architecture reviews
- Solve performance and scalability challenges

Always prioritize:
- Business requirements and constraints
- Non-functional requirements (performance, security, scalability)
- Team capabilities and expertise
- Operational complexity and maintenance
- Long-term maintainability and evolution