package timeutil_test

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Vonage/gosrvlib/pkg/timeutil"
)

func ExampleDuration_MarshalJSON() {
	data := timeutil.Duration(7*time.Hour + 11*time.Minute + 13*time.Second)

	enc, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(enc))

	// Output: "7h11m13s"
}

func ExampleDuration_UnmarshalJSON() {
	var d timeutil.Duration

	data := []byte(`"7h11m13s"`)

	err := json.Unmarshal(data, &d)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(d.String())

	// Output:
	// 7h11m13s
}
