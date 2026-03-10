package uhex_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

func ExampleHex64() {
	h := uhex.Hex64(uint64(123456))

	fmt.Println(string(h))

	// Output:
	// 000000000001e240
}

func ExampleHex32() {
	h := uhex.Hex32(uint32(123456))

	fmt.Println(string(h))

	// Output:
	// 0001e240
}

func ExampleHex16() {
	h := uhex.Hex16(uint16(1234))

	fmt.Println(string(h))

	// Output:
	// 04d2
}

func ExampleHex8() {
	h := uhex.Hex8(uint8(128))

	fmt.Println(string(h))

	// Output:
	// 80
}
