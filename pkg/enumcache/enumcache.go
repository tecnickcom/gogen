/*
Package enumcache provides thread-safe storage and lookup for enumeration name
and ID mappings.

It solves the problem of maintaining reusable bidirectional enum mappings in
concurrent applications, including support for bitmask-based enum sets.

The cache is useful when you need fast mapping from string names to integer IDs
and back again. It also supports encoding and decoding enum bitmaps when enum
IDs represent bit flags.

Typical usage is set-once, read-many: populate entries during startup using Set,
SetAllIDByName, or SetAllNameByID, then perform lookups from application code.

Top features:

  - concurrent-safe name-to-ID and ID-to-name lookup with sync.RWMutex protection
  - bulk population helpers for loading enum definitions from code or external sources
  - deterministic sorted retrieval with SortNames and SortIDs for logs, output, and tests
  - binary-map encoding and decoding via github.com/tecnickcom/gogen/pkg/enumbitmap
    for flag-style enum values
  - explicit error returns when IDs or names are missing, improving caller control

Benefits:

  - keep enum mappings centralized and thread-safe
  - reduce duplication of lookup logic across services
  - simplify support for feature flags and bitmask enums
*/
package enumcache

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tecnickcom/gogen/pkg/enumbitmap"
)

// IDByName maps enum names to numeric IDs.
type IDByName map[string]int

// NameByID maps integers to string names.
type NameByID map[int]string

// EnumCache stores bidirectional enum mappings (name<->ID).
type EnumCache struct {
	sync.RWMutex

	id   IDByName
	name NameByID
}

// New creates an empty thread-safe enum cache.
//
// The cache supports bidirectional name/id lookups and bitmask conversions.
func New() *EnumCache {
	return &EnumCache{
		id:   make(IDByName),
		name: make(NameByID),
	}
}

// Set stores a single enum mapping pair.
//
// Existing values for id or name are overwritten.
func (ec *EnumCache) Set(id int, name string) {
	ec.Lock()
	defer ec.Unlock()

	ec.name[id] = name
	ec.id[name] = id
}

// SetAllIDByName bulk-loads enum values from name-to-id input.
//
// It is useful when parsing static definitions keyed by symbolic names.
func (ec *EnumCache) SetAllIDByName(enum IDByName) {
	ec.Lock()
	defer ec.Unlock()

	for name, id := range enum {
		ec.name[id] = name
		ec.id[name] = id
	}
}

// SetAllNameByID bulk-loads enum values from id-to-name input.
//
// It is useful when loading rows from storage keyed by numeric IDs.
func (ec *EnumCache) SetAllNameByID(enum NameByID) {
	ec.Lock()
	defer ec.Unlock()

	for id, name := range enum {
		ec.name[id] = name
		ec.id[name] = id
	}
}

// ID returns the numeric ID associated with name.
//
// It returns an error when name is not present.
func (ec *EnumCache) ID(name string) (int, error) {
	ec.RLock()
	defer ec.RUnlock()

	id, ok := ec.id[name]
	if !ok {
		return 0, fmt.Errorf("cache name not found: %s", name)
	}

	return id, nil
}

// Name returns the symbolic name associated with id.
//
// It returns an error when id is not present.
func (ec *EnumCache) Name(id int) (string, error) {
	ec.RLock()
	defer ec.RUnlock()

	name, ok := ec.name[id]
	if !ok {
		return "", fmt.Errorf("cache ID not found: %d", id)
	}

	return name, nil
}

// SortNames returns all cached names in ascending lexical order.
//
// This is useful for deterministic output and tests.
func (ec *EnumCache) SortNames() []string {
	ec.RLock()
	defer ec.RUnlock()

	sorted := make([]string, 0, len(ec.id))
	for name := range ec.id {
		sorted = append(sorted, name)
	}

	sort.Strings(sorted)

	return sorted
}

// SortIDs returns all cached IDs in ascending numeric order.
//
// This is useful for deterministic output and tests.
func (ec *EnumCache) SortIDs() []int {
	ec.RLock()
	defer ec.RUnlock()

	sorted := make([]int, 0, len(ec.name))
	for id := range ec.name {
		sorted = append(sorted, id)
	}

	sort.Ints(sorted)

	return sorted
}

// DecodeBinaryMap expands a bitmask value into enum names.
//
// The cache must contain bit-value IDs (1<<n) mapped to names.
func (ec *EnumCache) DecodeBinaryMap(v int) ([]string, error) {
	ec.RLock()
	defer ec.RUnlock()

	return enumbitmap.BitMapToStrings(ec.name, v) //nolint:wrapcheck
}

// EncodeBinaryMap combines enum names into a bitmask value.
//
// The cache must contain bit-value IDs (1<<n) mapped to names.
func (ec *EnumCache) EncodeBinaryMap(s []string) (int, error) {
	ec.RLock()
	defer ec.RUnlock()

	return enumbitmap.StringsToBitMap(ec.id, s) //nolint:wrapcheck
}
