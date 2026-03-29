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
	// digit characters and no match was found.
	StatusMatchEmpty int8 = -127 // 0b10000001

	// StatusMatchNo indicates that no match was found because the first digit
	// of the input does not correspond to any child of the trie root.
	StatusMatchNo int8 = -125 // 0b10000011

	// StatusMatchFull indicates an exact match: every digit of the input was
	// consumed and the final trie node is a leaf (no children).
	StatusMatchFull int8 = 0 // 0b00000000

	// StatusMatchPartial indicates that every digit of the input was consumed
	// and the final trie node is not a leaf (it has children). The input is a
	// prefix of at least one longer stored key.
	StatusMatchPartial int8 = 1 // 0b00000001

	// StatusMatchPrefix indicates that the trie path was exhausted before all
	// input digits were consumed: a stored key is a prefix of the input. The
	// value at the last matched node (a leaf) is returned.
	StatusMatchPrefix int8 = 2 // 0b00000010

	// StatusMatchPartialPrefix indicates that the trie path was exhausted
	// before all input digits were consumed and the last matched node is not a
	// leaf. The last non-nil value found along the path is returned.
	StatusMatchPartialPrefix int8 = 3 // 0b00000011
)

// indexSize is the number of possible children for each trie node.
const indexSize = 10 // digits from 0 to 9

// Node is a generic numerical-indexed trie node that stores a value of type T.
//
// Each node holds up to 10 children, one per digit 0–9. Non-digit characters
// in keys are skipped during traversal, making the trie tolerant of formatted
// numbers (e.g. "+1-800-555-0100") and vanity letter sequences.
//
// The zero value is not usable; create a root node with [New].
type Node[T any] struct {
	value       *T
	numChildren int
	children    [indexSize]*Node[T]
}

// New creates and returns an empty root [Node][T].
func New[T any]() *Node[T] {
	return &Node[T]{}
}

// Add stores val in the trie at the position defined by the digit sequence in
// num. Non-digit characters and separators in num are silently skipped;
// letter characters are mapped to their ITU E.161 keypad digits.
//
// Returns true if the key is new (no previous value was stored at that
// position), or false if an existing non-nil value was overwritten.
func (t *Node[T]) Add(num string, val *T) bool {
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

// Get retrieves a value from the trie using longest-prefix match on num.
//
// The trie is traversed digit by digit. Non-digit characters in num are
// skipped. Get returns the last non-nil value encountered on the path (the
// deepest matching prefix), together with a status code that describes the
// quality of the match:
//
//   - [StatusMatchEmpty]         (-127) — no digits in input
//   - [StatusMatchNo]            (-125) — no root child for first digit
//   - [StatusMatchFull]          (   0) — exact match, leaf node
//   - [StatusMatchPartial]       (   1) — exact match, non-leaf node
//   - [StatusMatchPrefix]        (   2) — stored key is a prefix of input
//   - [StatusMatchPartialPrefix] (   3) — stored key is a prefix of input, non-leaf
//
// The returned pointer may be nil when no value was set on any node along the
// matched path (e.g. intermediate nodes with no stored value). Always check
// for nil before dereferencing.
func (t *Node[T]) Get(num string) (*T, int8) {
	var match, digit int

	node := t
	val := node.value // the root node value is also the default value

	for _, v := range num {
		i, ok := phonekeypad.KeypadDigit(v)
		if !ok {
			// ingnore non-digit characters
			continue
		}

		digit++

		if node.children[i] == nil {
			// there are no more children to match
			if node.value != nil {
				val = node.value
			}

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

	status := (int8(typeutil.BoolToInt(match == 0)<<7) |
		int8(typeutil.BoolToInt(digit > match)<<1) |
		int8(typeutil.BoolToInt(node.numChildren > 0)))

	return val, status
}
