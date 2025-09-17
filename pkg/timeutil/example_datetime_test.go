package timeutil_test

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Vonage/gosrvlib/pkg/timeutil"
)

func ExampleDateTime_MarshalJSON() {
	dt := timeutil.DateTime[timeutil.TRFC3339](time.Date(2023, 1, 2, 15, 4, 5, 0, time.UTC))

	b, err := json.Marshal(dt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output: "2023-01-02T15:04:05Z"
}

func ExampleDateTime_UnmarshalJSON() {
	var dt timeutil.DateTime[timeutil.TRFC3339]

	data := []byte(`"2023-01-02T15:04:05Z"`)

	err := json.Unmarshal(data, &dt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(dt.String())

	// Output: 2023-01-02T15:04:05Z
}
