package countryphone_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/countryphone"
)

func ExampleData_NumberType() {
	// load defaut data
	data := countryphone.New(nil)

	info, err := data.NumberInfo("1357123456")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// {
	//   "type": 1,
	//   "geo": [
	//     {
	//       "alpha2": "US",
	//       "area": "California",
	//       "type": 1
	//     }
	//   ]
	// }
}
