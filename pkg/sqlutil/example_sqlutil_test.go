package sqlutil_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/sqlutil"
)

func ExampleSQLUtil_QuoteID() {
	q, err := sqlutil.New()
	if err != nil {
		log.Fatal(err)
	}

	o := q.QuoteID("7919")

	fmt.Println(o)

	// Output:
	// `7919`
}

func ExampleWithQuoteIDFunc() {
	// define custom quote function
	fn := func(s string) string { return "TEST-" + s }

	q, err := sqlutil.New(
		sqlutil.WithQuoteIDFunc(fn),
	)
	if err != nil {
		log.Fatal(err)
	}

	o := q.QuoteID("6971")

	fmt.Println(o)

	// Output:
	// TEST-6971
}

func ExampleSQLUtil_QuoteValue() {
	q, err := sqlutil.New()
	if err != nil {
		log.Fatal(err)
	}

	o := q.QuoteValue("5867")

	fmt.Println(o)

	// Output:
	// '5867'
}

func ExampleWithQuoteValueFunc() {
	// define custom quote function
	fn := func(s string) string { return "TEST-" + s }

	q, err := sqlutil.New(
		sqlutil.WithQuoteValueFunc(fn),
	)
	if err != nil {
		log.Fatal(err)
	}

	o := q.QuoteValue("4987")

	fmt.Println(o)

	// Output:
	// TEST-4987
}
