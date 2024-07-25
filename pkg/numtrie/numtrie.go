/*
Package numtrie provides a numerical-indexed trie data structure. The trie
allows to store and retrieve values of any type associated with a numerical key.
It supports partial matches and alphabetical phone numbers.
*/
package numtrie

import (
	"github.com/Vonage/gosrvlib/pkg/phonekeypad"
	"github.com/Vonage/gosrvlib/pkg/typeutil"
)

// Status codes to be returned when searching for a number in the trie.
const (
	// StatusMatchEmpty indicates that the input string is empty and no match
	// was found.
	StatusMatchEmpty int8 = -127 // 0b10000001

	// StatusMatchNo indicates that no match was found. The first number digit
	// doesn't match any value at the trie root.
	StatusMatchNo int8 = -125 // 0b10000011

	// StatusMatchFull indicates that a full exact match was found. The full
	// number matches a trie leaf.
	StatusMatchFull int8 = 0 // 0b00000000

	// StatusMatchPartial indicates that the full number matches a trie node
	// that is not a leaf.
	StatusMatchPartial int8 = 1 // 0b00000001

	// StatusMatchPrefix indicates that only a prefix of the number matches a
	// trie leaf. The remaining digits are not present in the trie.
	StatusMatchPrefix int8 = 2 // 0b00000010

	// StatusMatchPartialPrefix indicates that only a prefix of the number
	// matches a trie node that is not a leaf. The remaining digits are not
	// present in the trie.
	StatusMatchPartialPrefix int8 = 3 // 0b00000011
)

const indexSize = 10 // digits from 0 to 9

// Node is a numerical-indexed trie node that stores a value of any type.
type Node[T any] struct {
	value       *T
	numChildren int
	children    [indexSize]*Node[T]
}

// New creates a new Node.
func New[T any]() *Node[T] {
	return &Node[T]{}
}

// Add adds a value to the trie with the given numerical key.
// It returns false if the value overrides an existing non-nil one.
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

// Get retrieves a value from the trie with the given numerical key. It supports
// partial matches. It returns the last non-nil value found in the trie path for
// the specified number. The return value should always be checked for the nil
// value. The second return value provides information about the match status:
//   - StatusMatchEmpty (-127 = 0b10000001) indicates that the input string is
//     empty and no match was found.
//   - StatusMatchNo (-125 = 0b10000011) indicates that no match was found. The
//     first number digit doesn't match any value at the trie root.
//   - StatusMatchFull (0 = 0b00000000) indicates that a full exact match was
//     found. The full number matches a trie leaf.
//   - StatusMatchPartial (1 = 0b00000001) indicates that the full number matches
//     a trie node that is not a leaf.
//   - StatusMatchPrefix (2 = 0b00000010) indicates that only a prefix of the
//     number matches a trie leaf. The remaining digits are not present in the trie.
//   - StatusMatchPartialPrefix (3 = 0b00000011) indicates that only a prefix of
//     the number matches a trie node that is not a leaf. The remaining digits are
//     not present in the trie.
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
