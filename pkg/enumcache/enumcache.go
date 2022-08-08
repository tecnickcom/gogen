// Package enumcache provides simple methods to store and retrieve enumerations with a numerical ID and string name.
package enumcache

import (
	"fmt"
	"sort"
	"sync"

	"github.com/nexmoinc/gosrvlib/pkg/enumbitmap"
)

// IDByName type maps strings to integers IDs.
type IDByName map[string]int

// NameByID maps integers to string names.
type NameByID map[int]string

// EnumCache handles name and id value mapping.
type EnumCache struct {
	sync.RWMutex
	id   IDByName
	name NameByID
}

// New returns a new empty EnumCache.
func New() *EnumCache {
	return &EnumCache{
		id:   make(IDByName),
		name: make(NameByID),
	}
}

// Set a single id-name key-value.
func (ec *EnumCache) Set(id int, name string) {
	ec.Lock()
	defer ec.Unlock()

	ec.name[id] = name
	ec.id[name] = id
}

// SetAllIDByName sets all the specified enumeration ID values indexed by Name.
func (ec *EnumCache) SetAllIDByName(enum IDByName) {
	ec.Lock()
	defer ec.Unlock()

	for name, id := range enum {
		ec.name[id] = name
		ec.id[name] = id
	}
}

// SetAllNameByID sets all the specified enumeration Name values indexed by ID.
func (ec *EnumCache) SetAllNameByID(enum NameByID) {
	ec.Lock()
	defer ec.Unlock()

	for id, name := range enum {
		ec.name[id] = name
		ec.id[name] = id
	}
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

// DecodeBinaryMap decodes a int binary map into a list of string names.
// The EnumCache must contain the mapping between the bit values and the names.
func (ec *EnumCache) DecodeBinaryMap(v int) (s []string, err error) {
	return enumbitmap.BitMapToStrings(ec.name, v) //nolint:wrapcheck
}

// EncodeBinaryMap encode a list of string names into a int binary map.
// The EnumCache must contain the mapping between the bit values and the names.
func (ec *EnumCache) EncodeBinaryMap(s []string) (v int, err error) {
	return enumbitmap.StringsToBitMap(ec.id, s) //nolint:wrapcheck
}
