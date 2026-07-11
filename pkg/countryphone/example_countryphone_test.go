package countryphone_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/tecnickcom/nurago/pkg/countryphone"
)

func ExampleData_NumberInfo() {
	// load default data
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

func ExampleData_NumberType() {
	// load default data
	data := countryphone.New(nil)

	label, err := data.NumberType(2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(label)

	// Output:
	// mobile
}

func ExampleData_AreaType() {
	// load default data
	data := countryphone.New(nil)

	label, err := data.AreaType(1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(label)

	// Output:
	// state
}
