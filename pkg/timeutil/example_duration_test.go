package timeutil_test

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Vonage/gosrvlib/pkg/timeutil"
)

func ExampleDuration_MarshalJSON() {
	type testData struct {
		Time timeutil.Duration `json:"Time"`
	}

	data := testData{
		Time: timeutil.Duration(7*time.Hour + 11*time.Minute + 13*time.Second),
	}

	enc, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(enc))

	// Output:
	// {"Time":"7h11m13s"}
}

func ExampleDuration_UnmarshalJSON() {
	type testData struct {
		Time timeutil.Duration `json:"Time"`
	}

	enc := []byte(`{"Time":"7h11m13s"}`)

	var data testData

	err := json.Unmarshal(enc, &data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data.Time.String())

	// Output:
	// 7h11m13s
}
