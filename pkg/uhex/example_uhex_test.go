package uhex_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

func ExampleHex64() {
	h := uhex.Hex64(uint64(0x0123456789abcdef))

	fmt.Println(string(h))

	// Output:
	// 0123456789abcdef
}

func ExampleHex64UB() {
	dst := [16]byte{}
	uhex.Hex64UB(uint64(0x0123456789abcdef), &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// 0123456789abcdef
}

func ExampleHex64B() {
	h := uhex.Hex64B([8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef})

	fmt.Println(string(h))

	// Output:
	// 0123456789abcdef
}

func ExampleHex64BB() {
	dst := [16]byte{}
	uhex.Hex64BB([8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}, &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// 0123456789abcdef
}

func ExampleHex32() {
	h := uhex.Hex32(uint32(0x89abcdef))

	fmt.Println(string(h))

	// Output:
	// 89abcdef
}

func ExampleHex32UB() {
	dst := [8]byte{}
	uhex.Hex32UB(uint32(0x89abcdef), &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// 89abcdef
}

func ExampleHex32B() {
	h := uhex.Hex32B([4]byte{0x89, 0xab, 0xcd, 0xef})

	fmt.Println(string(h))

	// Output:
	// 89abcdef
}

func ExampleHex32BB() {
	dst := [8]byte{}
	uhex.Hex32BB([4]byte{0x89, 0xab, 0xcd, 0xef}, &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// 89abcdef
}

func ExampleHex16() {
	h := uhex.Hex16(uint16(0xabcd))

	fmt.Println(string(h))

	// Output:
	// abcd
}

func ExampleHex16UB() {
	dst := [4]byte{}
	uhex.Hex16UB(uint16(0xabcd), &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// abcd
}

func ExampleHex16B() {
	h := uhex.Hex16B([2]byte{0xab, 0xcd})

	fmt.Println(string(h))

	// Output:
	// abcd
}

func ExampleHex16BB() {
	dst := [4]byte{}
	uhex.Hex16BB([2]byte{0xab, 0xcd}, &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// abcd
}

func ExampleHex8() {
	h := uhex.Hex8(uint8(128))

	fmt.Println(string(h))

	// Output:
	// 80
}

func ExampleHex8UB() {
	dst := [2]byte{}
	uhex.Hex8UB(uint8(128), &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// 80
}

func ExampleHex8B() {
	h := uhex.Hex8B([1]byte{0xab})

	fmt.Println(string(h))

	// Output:
	// ab
}

func ExampleHex8BB() {
	dst := [2]byte{}
	uhex.Hex8BB([1]byte{0xab}, &dst)

	fmt.Println(string(dst[:]))

	// Output:
	// ab
}
