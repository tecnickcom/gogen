package enumcache_test

import (
	"fmt"
	"log"

	"github.com/vonage/gosrvlib/pkg/enumcache"
)

func ExampleNew() {
	// create a new cache
	ec := enumcache.New()

	// add an entry
	ec.Set(1, "alpha")

	// get the numerical ID associated to a string
	id, err := ec.ID("alpha")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id)

	// get the string name associated to a numerical ID
	name, err := ec.Name(1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(name)

	// Output:
	// 1
	// alpha
}

func ExampleEnumCache_SetAllIDByName() {
	// create a new cache
	ec := enumcache.New()

	// define cache entries indexed by string
	e := enumcache.IDByName{
		"first":  11,
		"second": 23,
		"third":  31,
	}

	// populate the cache with the specified entries
	ec.SetAllIDByName(e)

	// get the numerical ID associated to a string
	id, err := ec.ID("second")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id)

	// get the string name associated to a numerical ID
	name, err := ec.Name(23)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(name)

	// Output:
	// 23
	// second
}

func ExampleEnumCache_SetAllNameByID() {
	// create a new cache
	ec := enumcache.New()

	// define cache entries indexed by numerical ID
	e := enumcache.NameByID{
		11: "first",
		23: "second",
		31: "third",
	}

	// populate the cache with the specified entries
	ec.SetAllNameByID(e)

	// get the numerical ID associated to a string
	id, err := ec.ID("second")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id)

	// get the string name associated to a numerical ID
	name, err := ec.Name(23)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(name)

	// Output:
	// 23
	// second
}

func ExampleEnumCache_SortNames() {
	// create a new cache
	ec := enumcache.New()

	// define cache entries indexed by numerical ID
	e := enumcache.NameByID{
		1:  "delta",
		2:  "charlie",
		4:  "bravo",
		8:  "foxtrot",
		16: "echo",
		32: "alpha",
	}

	// populate the cache with the specified entries
	ec.SetAllNameByID(e)

	// get the sorted list of names
	sorted := ec.SortNames()

	fmt.Println(sorted)

	// Output:
	// [alpha bravo charlie delta echo foxtrot]
}

func ExampleEnumCache_SortIDs() {
	// create a new cache
	ec := enumcache.New()

	// define cache entries indexed by numerical ID
	e := enumcache.NameByID{
		55: "delta",
		33: "charlie",
		22: "bravo",
		66: "foxtrot",
		44: "echo",
		11: "alpha",
	}

	// populate the cache with the specified entries
	ec.SetAllNameByID(e)

	// get the sorted list of IDs
	sorted := ec.SortIDs()

	fmt.Println(sorted)

	// Output:
	// [11 22 33 44 55 66]
}

func ExampleEnumCache_DecodeBinaryMap() {
	// create a new cache
	ec := enumcache.New()

	ec.Set(0, "first")    // 00000000
	ec.Set(1, "second")   // 00000001
	ec.Set(2, "third")    // 00000010
	ec.Set(4, "fourth")   // 00000100
	ec.Set(8, "fifth")    // 00001000
	ec.Set(16, "sixth")   // 00010000
	ec.Set(32, "seventh") // 00100000
	ec.Set(64, "eighth")  // 01000000
	ec.Set(128, "ninth")  // 10000000

	// convert binary code to a slice of strings
	s, err := ec.DecodeBinaryMap(0b00101010) // 42
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(s)

	// Output:
	// [third fifth seventh]
}

func ExampleEnumCache_EncodeBinaryMap() {
	// create a new cache
	ec := enumcache.New()

	ec.Set(0, "first")    // 00000000
	ec.Set(1, "second")   // 00000001
	ec.Set(2, "third")    // 00000010
	ec.Set(4, "fourth")   // 00000100
	ec.Set(8, "fifth")    // 00001000
	ec.Set(16, "sixth")   // 00010000
	ec.Set(32, "seventh") // 00100000
	ec.Set(64, "eighth")  // 01000000
	ec.Set(128, "ninth")  // 10000000

	// convert a slice of string to the equivalent binary code
	v, err := ec.EncodeBinaryMap([]string{"third", "fifth", "seventh"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(v)

	// Output:
	// 42
}
