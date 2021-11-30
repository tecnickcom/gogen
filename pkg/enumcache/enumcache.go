// Package enumcache provides simple methods to store and retrieve enumerations with a numerical ID and string name.
package enumcache

import (
	"fmt"
	"sort"
	"sync"
)

// EnumCache handles name and id value mapping.
type EnumCache struct {
	sync.RWMutex
	id   map[string]int
	name map[int]string
}

// New returns a new empty EnumCache.
func New() *EnumCache {
	return &EnumCache{
		id:   make(map[string]int),
		name: make(map[int]string),
	}
}

// Set a single id-name key-value.
func (ec *EnumCache) Set(id int, name string) {
	ec.Lock()
	defer ec.Unlock()

	ec.name[id] = name
	ec.id[name] = id
}

// ID returns the numerical ID associated to the given name.
func (ec *EnumCache) ID(name string) (int, error) {
	ec.RLock()
	defer ec.RUnlock()

	id, ok := ec.id[name]
	if !ok {
		return 0, fmt.Errorf("cache name not found: %s", name)
	}

	return id, nil
}

// Name returns the name associated with the given numerical ID.
func (ec *EnumCache) Name(id int) (string, error) {
	ec.RLock()
	defer ec.RUnlock()

	name, ok := ec.name[id]
	if !ok {
		return "", fmt.Errorf("cache ID not found: %d", id)
	}

	return name, nil
}

// SortNames returns a list of sorted names.
func (ec *EnumCache) SortNames() []string {
	sorted := make([]string, 0, len(ec.id))
	for name := range ec.id {
		sorted = append(sorted, name)
	}

	sort.Strings(sorted)

	return sorted
}

// SortIDs returns a list of sorted IDs.
func (ec *EnumCache) SortIDs() []int {
	sorted := make([]int, 0, len(ec.name))
	for id := range ec.name {
		sorted = append(sorted, id)
	}

	sort.Ints(sorted)

	return sorted
}
