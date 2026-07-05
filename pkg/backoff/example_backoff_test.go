package backoff_test

import (
	"fmt"
	"time"

	"github.com/tecnickcom/gogen/pkg/backoff"
)

func ExampleSchedule() {
	s := backoff.New(backoff.Config{
		Base:     100 * time.Millisecond,
		Factor:   2,
		Jitter:   0, // disabled so the example output is deterministic
		MaxDelay: 350 * time.Millisecond,
	})

	for range 4 {
		fmt.Println(s.Next())
	}

	// Output:
	// 100ms
	// 200ms
	// 350ms
	// 350ms
}
