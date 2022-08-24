package filter_test

import (
	"fmt"
	"log"
	"net/url"

	"github.com/nexmoinc/gosrvlib/pkg/filter"
)

// Address is an example structure type used to test nested structures.
type Address struct {
	Country string `json:"country"`
}

// ID is an example structure type.
type ID struct {
	Name string  `json:"name"`
	Age  int     `json:"age"`
	Addr Address `json:"address"`
}

func ExampleProcessor_Apply() {
	// Simulate an encoded query passed in the http.Request of a http.Handler
	encodedJSONFilter := "%5B%5B%7B%22field%22%3A%22name%22%2C%22type%22%3A%22%3D%3D%22%2C%22value%22%3A%22doe%22%7D%2C%7B%22field%22%3A%22age%22%2C%22type%22%3A%22%3C%3D%22%2C%22value%22%3A42%7D%5D%2C%5B%7B%22field%22%3A%22address.country%22%2C%22type%22%3A%22regexp%22%2C%22value%22%3A%22%5EEN%24%7C%5EFR%24%22%7D%5D%5D"

	u, err := url.Parse("https://server.com/items?filter=" + encodedJSONFilter)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the filter with options
	// * WithJSONValues: We want to be lenient on the typing since we create the filter from JSON which handles a few types
	// * WithFieldNameTag: to express the filter based on JSON tags and not the actual field names
	f, err := filter.New(
		filter.WithFieldNameTag("json"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// The filter matches the following pretty printed json:
	//
	//	[
	//	  [
	//	    {
	//	      "field": "name",
	//	      "type": "==",
	//	      "value": "doe"
	//	    },
	//	    {
	//	      "field": "age",
	//	      "type": "<=",
	//	      "value": 42
	//	    }
	//	  ],
	//	  [
	//	    {
	//	      "field": "address.country",
	//	      "type": "regexp",
	//	      "value": "^EN$|^FR$"
	//	    }
	//	  ]
	//	]
	//
	// can be represented in one line as:
	//
	//	[[{"field":"name","type":"==","value":"doe"},{"field":"age","type":"<=","value":42}],[{"field":"address.country","type":"regexp","value":"^EN$|^FR$"}]]
	//
	// and URL-encoded as a query parameter:
	//
	//	filter=%5B%5B%7B%22field%22%3A%22name%22%2C%22type%22%3A%22%3D%3D%22%2C%22value%22%3A%22doe%22%7D%2C%7B%22field%22%3A%22age%22%2C%22type%22%3A%22%3C%3D%22%2C%22value%22%3A42%7D%5D%2C%5B%7B%22field%22%3A%22address.country%22%2C%22type%22%3A%22regexp%22%2C%22value%22%3A%22%5EEN%24%7C%5EFR%24%22%7D%5D%5D
	//
	// the equivalent logic is:
	//
	//	((name==doe OR age<=42) AND (address.country match "EN" or "FR"))
	//
	// It means that either the name OR the age must match exactly AND the country must match its regular expression.
	rules, err := f.ParseURLQuery(u.Query())
	if err != nil {
		log.Fatal(err)
	}

	// Given this list, the last item will be filtered
	list := []ID{
		{
			Name: "doe",
			Age:  55,
			Addr: Address{
				Country: "EN",
			},
		},
		{
			Name: "dupont",
			Age:  42,
			Addr: Address{
				Country: "FR",
			},
		},
		{
			Name: "doe",
			Age:  41,
			Addr: Address{
				Country: "US",
			},
		},
	}

	// Filters the list in place
	sliceLen, totalMatches, err := f.Apply(rules, &list)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(sliceLen)
	fmt.Println(totalMatches)

	for _, id := range list {
		fmt.Println(id)
	}

	// Output:
	// 2
	// 2
	// {doe 55 {EN}}
	// {dupont 42 {FR}}
}
