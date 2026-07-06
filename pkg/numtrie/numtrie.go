/*
Package numtrie provides a generic, digit-indexed [trie] (prefix tree) for
associating values of any type with numerical keys, with built-in support for
partial/prefix matching and alphabetical (vanity) phone-number keys.

# Problem

Telephony routing tables, dial-plan engines, number-classification services,
and similar systems must map a dialed number to a route, tariff, or carrier
by walking from the most-specific prefix to the least-specific one. A hash map
cannot express this longest-prefix-match semantics efficiently. A naive scan
over a sorted list degrades at scale. A trie provides O(k) lookup (where k is
the number of digits) with natural support for prefix traversal and partial
matches.

# Solution

[Node] is a generic trie node parameterised on the value type. Build the trie
once with [Node.Add], then query it repeatedly with [Node.Get]:

	// Build a routing table.
	root := numtrie.New[Route]()
	root.Add("1",      &defaultUSRoute)
	root.Add("1212",   &newYorkRoute)
	root.Add("44",     &ukRoute)
	root.Add("44207",  &londonRoute)

	// Longest-prefix lookup for an incoming call.
	val, status := root.Get("+1-212-555-0100")
	if val != nil {
		// val == &newYorkRoute (longest matching prefix: "1212")
	}

# Features

  - Generic value type: [Node][T] works with any type T; no interface{}
    assertions or separate value maps required.
  - Longest-prefix / partial-match semantics: [Node.Get] returns the last
    non-nil value found while walking the trie path, implementing longest-prefix
    match in a single traversal.
  - Exact-match lookup: [Node.GetExact] returns the value stored exactly at a
    key position, without any longest-prefix fallback.
  - Six fine-grained match status codes: the int8 status returned by [Node.Get]
    distinguishes empty input, no match, full exact match, partial match (input
    is a prefix of a stored key), prefix match (stored key is a prefix of the
    input), and the combined partial-prefix case — giving callers precise
    information for routing decisions without a second lookup.
  - Separator-tolerant keys: non-digit characters (hyphens, spaces,
    parentheses, '+') are silently ignored during both [Node.Add] and
    [Node.Get], so E.164 formatted numbers like "+1-212-555-0100" work
    without sanitisation.
  - Alphabetical key support: letter characters in keys are converted to
    their ITU E.161 phone-keypad digits via
    [github.com/tecnickcom/gogen/pkg/phonekeypad], enabling vanity numbers
    like "1-800-FLOWERS" to be stored and matched transparently.
  - Efficient memory layout: each node holds a fixed 10-slot children array
    (one slot per digit 0–9), avoiding dynamic allocation per insertion.

# Match Status Codes

The status int8 returned by [Node.Get] is a compact bit field:

	Bit 7 (sign): set   → no digits matched at all (empty input or no root child)
	Bit 1:        set   → input extends beyond the matched trie path (prefix match)
	Bit 0:        set   → matched node has children (partial match)

The six named constants encode every meaningful combination:

	StatusMatchEmpty         (-127) — input was empty
	StatusMatchNo            (-125) — first digit not in trie
	StatusMatchFull          (   0) — exact match, leaf node
	StatusMatchPartial       (   1) — exact match, non-leaf node
	StatusMatchPrefix        (   2) — stored key is prefix of input, leaf node
	StatusMatchPartialPrefix (   3) — stored key is prefix of input, non-leaf node

When bit 7 is set the status is a standalone sentinel (StatusMatchEmpty or
StatusMatchNo): only bit 7 is significant, the low bits carry no meaning, and
[Node.Get] returns a nil value for these statuses even when a root/default value
is present.

# Concurrency

A [Node] is not safe for concurrent modification: [Node.Add] mutates the trie in
place. Once the trie is fully built it may be queried concurrently by any number
of goroutines via [Node.Get] and [Node.GetExact], provided no [Node.Add] runs
concurrently.

# Benefits

This package delivers O(k) longest-prefix match over numerical keys in a
single import, replacing ad-hoc scan loops with a purpose-built data structure
that naturally handles separators, vanity numbers, and fine-grained match
reporting.

[trie]: https://en.wikipedia.org/wiki/Trie
*/
package numtrie

import (
	"github.com/tecnickcom/gogen/pkg/phonekeypad"
	"github.com/tecnickcom/gogen/pkg/typeutil"
)

// Status codes to be returned when searching for a number in the trie.
const (
	// StatusMatchEmpty indicates that the input string contained no recognizable
	// digit characters and no match was found. [Node.Get] returns a nil value
	// for this status, even when a root/default value is present.
	StatusMatchEmpty int8 = -127 // 0b10000001

	// StatusMatchNo indicates that no match was found because the first digit
	// of the input does not correspond to any child of the trie root. [Node.Get]
	// returns a nil value for this status, even when a root/default value is
	// present.
	StatusMatchNo int8 = -125 // 0b10000011

	// StatusMatchFull indicates an exact match: every digit of the input was
	// consumed and the final trie node is a leaf (no children).
	StatusMatchFull int8 = 0 // 0b00000000

	// StatusMatchPartial indicates that every digit of the input was consumed
	// and the final trie node is not a leaf (it has children). The input is a
	// prefix of at least one longer stored key.
	StatusMatchPartial int8 = 1 // 0b00000001

	// StatusMatchPrefix indicates that the trie path was exhausted before all
	// input digits were consumed: a stored key is a prefix of the input. The last
	// non-nil value found along the path is returned (normally the value at the
	// matched leaf).
	StatusMatchPrefix int8 = 2 // 0b00000010

	// StatusMatchPartialPrefix indicates that the trie path was exhausted
	// before all input digits were consumed and the last matched node is not a
	// leaf. The last non-nil value found along the path is returned.
	StatusMatchPartialPrefix int8 = 3 // 0b00000011
)

// indexSize is the number of possible children for each trie node. It matches
// the [0, 10) range guaranteed by [github.com/tecnickcom/gogen/pkg/phonekeypad.KeypadDigit],
// whose result indexes the children array directly; the two must stay in sync.
const indexSize = 10 // digits from 0 to 9

// Node is a generic numerical-indexed trie node that stores a value of type T.
//
// Each node holds up to 10 children, one per digit 0–9. Non-digit characters
// in keys are skipped during traversal, making the trie tolerant of formatted
// numbers (e.g. "+1-800-555-0100") and vanity letter sequences.
//
// The zero value is not usable; create a root node with [New].
//
// A Node is not safe for concurrent modification: [Node.Add] mutates the trie in
// place. Once the trie is fully built it may be queried concurrently by any
// number of goroutines via [Node.Get] and [Node.GetExact], provided no
// [Node.Add] runs concurrently.
type Node[T any] struct {
	value       *T
	numChildren int
	children    [indexSize]*Node[T]
}

// New constructs an empty root Node[T] for building a numerical prefix trie.
func New[T any]() *Node[T] {
	return &Node[T]{}
}

// Add stores val at the trie position defined by num.
// Non-digit characters are skipped and letters are mapped to keypad digits.
// It returns true when the key was new, or false when an existing value was
// overwritten.
//
// Storing a value at the empty key (Add("", v)) sets a default returned by
// [Node.Get] as the longest-prefix fallback for any input that matches at least
// one digit.
//
// A nil val is rejected as a no-op: the trie is left unchanged and Add returns
// false, since the trie cannot store or distinguish a nil value from an absent
// one.
func (t *Node[T]) Add(num string, val *T) bool {
	if val == nil {
		return false
	}

	node := t

	for _, v := range num {
		i, ok := phonekeypad.KeypadDigit(v)
		if !ok {
			continue
		}

		if node.children[i] == nil {
			node.children[i] = New[T]()
			node.numChildren++
		}

		node = node.children[i]
	}

	isnew := (node.value == nil)

	node.value = val

	return isnew
}

// Get retrieves the longest-prefix match for num.
// Non-digit characters are skipped. The returned status indicates exact, partial,
// prefix, or no-match outcomes as described by the StatusMatch* constants.
//
// The status int8 returned by [Node.Get] is a compact bit field:
//
//	Bit 7 (sign): set   → no digits matched at all (empty input or no root child)
//	Bit 1:        set   → input extends beyond the matched trie path (prefix match)
//	Bit 0:        set   → matched node has children (partial match)
//
// The six named constants encode every meaningful combination:
//
//	StatusMatchEmpty         (-127) — input was empty
//	StatusMatchNo            (-125) — first digit not in trie
//	StatusMatchFull          (   0) — exact match, leaf node
//	StatusMatchPartial       (   1) — exact match, non-leaf node
//	StatusMatchPrefix        (   2) — stored key is prefix of input, leaf node
//	StatusMatchPartialPrefix (   3) — stored key is prefix of input, non-leaf node
//
// For the two negative sentinels (StatusMatchEmpty and StatusMatchNo) the
// returned value is nil, even when a root/default value is present.
func (t *Node[T]) Get(num string) (*T, int8) {
	var match, digit int

	node := t
	val := node.value // the root node value is the empty-prefix (default) match

	for _, v := range num {
		i, ok := phonekeypad.KeypadDigit(v)
		if !ok {
			// ignore non-digit characters
			continue
		}

		digit++

		if node.children[i] == nil {
			// no child for this digit: stop. val already holds the last non-nil
			// value seen on the path, including the current node.
			break
		}

		// move to the next child node
		node = node.children[i]

		if node.value != nil {
			// remember the last non-nil value found
			val = node.value
		}

		match++
	}

	status := node.matchStatus(match, digit)

	if match == 0 {
		// Pure no-match (empty input or first digit absent from the trie):
		// report no value, even when a root/default value is present.
		return nil, status
	}

	return val, status
}

// matchStatus derives the Get status code for a traversal that ended on the
// receiver node, having matched match digits out of digit input digits.
func (t *Node[T]) matchStatus(match, digit int) int8 {
	if match == 0 {
		// No digit was matched: return the named no-match constants directly,
		// so the status is well-defined even when the trie is empty.
		if digit == 0 {
			return StatusMatchEmpty
		}

		return StatusMatchNo
	}

	return typeutil.BoolToNum[int8](digit > match)<<1 |
		typeutil.BoolToNum[int8](t.numChildren > 0)
}

// GetExact retrieves the value stored at the exact trie position defined by num.
// Non-digit characters are skipped and letters are mapped to keypad digits,
// mirroring [Node.Add]. Unlike [Node.Get], it performs no longest-prefix
// fallback: it returns nil when the position does not exist in the trie or
// when no value was stored exactly at that position.
func (t *Node[T]) GetExact(num string) *T {
	node := t

	for _, v := range num {
		i, ok := phonekeypad.KeypadDigit(v)
		if !ok {
			// ignore non-digit characters
			continue
		}

		if node.children[i] == nil {
			return nil
		}

		node = node.children[i]
	}

	return node.value
}
