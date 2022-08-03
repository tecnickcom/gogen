package enumbitmap_test

import (
	"fmt"
	"log"

	"github.com/nexmoinc/gosrvlib/pkg/enumbitmap"
)

func ExampleBitMapToStrings() {
	// create a binary map
	// each bit correspond to a different entry
	eis := map[int]string{
		0:   "first",   // 00000000
		1:   "second",  // 00000001
		2:   "third",   // 00000010
		4:   "fourth",  // 00000100
		8:   "fifth",   // 00001000
		16:  "sixth",   // 00010000
		32:  "seventh", // 00100000
		64:  "eighth",  // 01000000
		128: "ninth",   // 10000000
	}

	// convert binary code to a slice of strings
	s, err := enumbitmap.BitMapToStrings(eis, 0b00101010) // 42
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(s)

	// Output:
	// [third fifth seventh]
}

func ExampleStringsToBitMap() {
	// create a binary map
	// each entry is assigned to a different bit
	esi := map[string]int{
		"first":   0,   // 00000000
		"second":  1,   // 00000001
		"third":   2,   // 00000010
		"fourth":  4,   // 00000100
		"fifth":   8,   // 00001000
		"sixth":   16,  // 00010000
		"seventh": 32,  // 00100000
		"eighth":  64,  // 01000000
		"ninth":   128, // 10000000
	}

	// convert a slice of string to the equivalent binary code
	b, err := enumbitmap.StringsToBitMap(
		esi,
		[]string{
			"third",
			"fifth",
			"seventh",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(b)

	// Output:
	// 42
}
