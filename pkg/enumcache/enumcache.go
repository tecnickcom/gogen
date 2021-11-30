// Package enumcache provides simple methods to store and retrieve enumerations with a numerical ID and string name.
package enumcache

import (
	"fmt"
	"sync"
)

// EnumCache handles name and id value mapping.
type EnumCache struct {
	sync.RWMutex
	id   map[string]int
	name map[int]string
}

// MakeEnumCache returns a new empty EnumCache.
func MakeEnumCache() *EnumCache {
	return &EnumCache{
		id:   make(map[string]int),
		name: make(map[int]string),
	}
}

// Set a single id-name key-value.
func (tc *EnumCache) Set(id int, name string) {
	tc.Lock()
	defer tc.Unlock()

	tc.name[id] = name
	tc.id[name] = id
}

// ID returns the numerical ID associated to the given name.
func (tc *EnumCache) ID(name string) (int, error) {
	tc.RLock()
	defer tc.RUnlock()

	id, ok := tc.id[name]
	if !ok {
		return 0, fmt.Errorf("cache name not found: %s", name)
	}

	return id, nil
}

// Name returns the name associated with the given numerical ID.
func (tc *EnumCache) Name(id int) (string, error) {
	tc.RLock()
	defer tc.RUnlock()

	name, ok := tc.name[id]
	if !ok {
		return "", fmt.Errorf("cache ID not found: %d", id)
	}

	return name, nil
}
