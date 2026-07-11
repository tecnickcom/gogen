package enumbitmap_test

import (
	"fmt"
	"log"

	"github.com/tecnickcom/nurago/pkg/enumbitmap"
)

func ExampleBitMapToStrings() {
	// create a binary map
	// each bit correspond to a different entry (IDs are single-bit powers of two)
	eis := map[int]string{
		1:   "first",   // 00000001
		2:   "second",  // 00000010
		4:   "third",   // 00000100
		8:   "fourth",  // 00001000
		16:  "fifth",   // 00010000
		32:  "sixth",   // 00100000
		64:  "seventh", // 01000000
		128: "eighth",  // 10000000
	}

	// convert binary code to a slice of strings
	s, err := enumbitmap.BitMapToStrings(eis, 0b00101010) // 42
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(s)

	// Output:
	// [second fourth sixth]
}

func ExampleStringsToBitMap() {
	// create a binary map
	// each entry is assigned to a different bit (single-bit powers of two)
	esi := map[string]int{
		"first":   1,   // 00000001
		"second":  2,   // 00000010
		"third":   4,   // 00000100
		"fourth":  8,   // 00001000
		"fifth":   16,  // 00010000
		"sixth":   32,  // 00100000
		"seventh": 64,  // 01000000
		"eighth":  128, // 10000000
	}

	// convert a slice of string to the equivalent binary code
	b, err := enumbitmap.StringsToBitMap(
		esi,
		[]string{
			"second",
			"fourth",
			"sixth",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(b)

	// Output:
	// 42
}
