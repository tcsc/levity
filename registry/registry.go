package registry

import (
	"sync"

	"github.com/google/uuid"
	"github.com/tcsc/levity/task"
)

// Registry models a simple task registration system. For a simple exercise
// like this we are just keeping the task information in memory. This would
// obviously require something more durable in a production server.
type Registry struct {
	lock sync.RWMutex
	db   map[string]*task.Task
}

func handleFromUUID() string {
	return uuid.New().String()
}

// New creates and returns an initialised, ready-to-use task registry
func New() *Registry {
	return &Registry{
		db: make(map[string]*task.Task),
	}
}

// Lookup attempts to find a task from its ID. Returns nil if no such task
// exists.
func (registry *Registry) Lookup(handle string) *task.Task {
	// TODO: Refactor the interface here to explicitly add an `ok`
	//       return value (in the style of reading a map[...]...)
	//       to differentiate a present nil value return vs. a
	//       not-present-at-all value.
	registry.lock.RLock()
	defer registry.lock.RUnlock()

	if t, exists := registry.db[handle]; exists {
		return t
	}

	return nil
}

// Register binda a task to a unique handle and registers it in the
// task database.
func (registry *Registry) Register(t *task.Task) string {
	// For the purposes of this exercise we're using a stringified UUID as
	// the task ID. This should be statistically unique - certainly we
	// should not expect to see a collision in the lifetime of the server.
	// It's reasonably safe to treat the generated handles as unique without
	// further checking. In fact, if we see duplicate handles we probably
	// have bigger problems on the system than this toy service misbehaving.

	handle := handleFromUUID()

	registry.lock.Lock()
	defer registry.lock.Unlock()

	registry.db[handle] = t
	return handle
}

// Len fetches the number of tasks stored in the registry
func (registry *Registry) Len() int {
	registry.lock.RLock()
	defer registry.lock.RUnlock()
	return len(registry.db)
}
