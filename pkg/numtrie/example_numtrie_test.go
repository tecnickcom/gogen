package numtrie_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/numtrie"
)

func ExampleNode_Get() {
	// create a new numerical-indexed trie that holds sting values
	node := numtrie.New[string]()

	valA := "gamma"
	node.Add("702", &valA)

	valB := "foxtrot"
	node.Add("702153", &valB)

	// StatusMatchEmpty (-127 = 0b10000001) indicates that the input string is
	// empty and no match was found.
	got, status := node.Get("")
	if got != nil {
		fmt.Println(*got, status)
	} else {
		fmt.Println(got, status)
	}

	// StatusMatchNo (-125 = 0b10000011) indicates that no match was found. The
	// first number digit doesn't match any value at the trie root.
	got, status = node.Get("111")
	if got != nil {
		fmt.Println(*got, status)
	} else {
		fmt.Println(got, status)
	}

	// StatusMatchFull (0 = 0b00000000) indicates that a full exact match was
	// found. The full number matches a trie leaf.
	got, status = node.Get("702153")
	if got != nil {
		fmt.Println(*got, status)
	}

	// StatusMatchPartial (1 = 0b00000001) indicates that the full number
	// matches a trie node that is not a leaf.
	got, status = node.Get("702")
	if got != nil {
		fmt.Println(*got, status)
	}

	// StatusMatchPrefix (2 = 0b00000010) indicates that only a prefix of the
	// number matches a trie leaf. The remaining digits are not present in the
	// trie.
	got, status = node.Get("702153-99")
	if got != nil {
		fmt.Println(*got, status)
	}

	// StatusMatchPartialPrefix (3 = 0b00000011) indicates that only a prefix of
	// the number matches a trie node that is not a leaf. The remaining digits
	// are not present in the trie.
	got, status = node.Get("702-99")
	if got != nil {
		fmt.Println(*got, status)
	}

	// StatusMatchPartialPrefix (3 = 0b00000011) indicates that only a prefix of
	// the number matches a trie node that is not a leaf. The remaining digits
	// are not present in the trie. The last non-nil value on the trie path is
	// returned.
	// The match is with 7021 but the node at 1 is nil, so the last non-nil
	// value at node 702 is returned.
	got, status = node.Get("702166")
	if got != nil {
		fmt.Println(*got, status)
	}

	// Output:
	// <nil> -127
	// <nil> -125
	// foxtrot 0
	// gamma 1
	// foxtrot 2
	// gamma 3
	// gamma 3
}
