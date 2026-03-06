package strsplit_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/strsplit"
)

func ExampleChunk() {
	str := "helloworld\nbellaciao"
	d := strsplit.Chunk(str, 5, 3)

	fmt.Println(d)

	// Output:
	// [hello world bella]
}

func ExampleChunkLine() {
	str := "hello,world"
	d := strsplit.ChunkLine(str, 8, -1)

	fmt.Println(d)

	// Output:
	// [hello, world]
}
