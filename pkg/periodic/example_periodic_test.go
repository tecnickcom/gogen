package periodic_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/periodic"
)

func ExampleNew() {
	count := make(chan int, 1)

	count <- 0

	// example task to execute periodically
	task := func(_ context.Context) {
		v := <-count
		count <- (v + 1)
	}

	interval := 10 * time.Millisecond
	jitter := 1 * time.Millisecond
	timeout := 1 * time.Millisecond

	// create a new periodic job
	p, err := periodic.New(interval, jitter, timeout, task)
	if err != nil {
		close(count)
		log.Fatal(err)
	}

	// start the periodic job
	p.Start(context.TODO())

	// wait for 3 times the interval
	wait := 3 * interval
	time.Sleep(wait)

	// stop the periodic job
	p.Stop()

	fmt.Println(<-count)

	close(count)

	// Output:
	// 3
}
