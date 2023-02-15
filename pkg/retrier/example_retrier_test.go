package retrier_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Vonage/gosrvlib/pkg/retrier"
)

func ExampleRetrier_Run() {
	var count int

	// example function that returns nil only at the third attempt.
	task := func(_ context.Context) error {
		if count == 2 {
			return nil
		}

		count++

		return fmt.Errorf("ERROR")
	}

	opts := []retrier.Option{
		retrier.WithRetryIfFn(retrier.DefaultRetryIf),
		retrier.WithAttempts(5),
		retrier.WithDelay(10 * time.Millisecond),
		retrier.WithDelayFactor(1.1),
		retrier.WithJitter(5 * time.Millisecond),
		retrier.WithTimeout(2 * time.Millisecond),
	}

	r, err := retrier.New(opts...)
	if err != nil {
		log.Fatal(err)
	}

	timeout := 1 * time.Second

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)

	err = r.Run(ctx, task)

	cancel()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(count)

	// Output:
	// 2
}
