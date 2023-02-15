package traceid_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Vonage/gosrvlib/pkg/traceid"
)

func ExampleNewContext() {
	// store value in context
	ctx := traceid.NewContext(context.Background(), "test-1-218549")

	// load the value from context and ignore default
	el1 := traceid.FromContext(ctx, "default-104173")

	fmt.Println(el1)

	// do not override the value in context
	ctx1 := traceid.NewContext(ctx, "test-2-563011")

	fmt.Println(ctx1)

	// Output:
	// test-1-218549
	// context.Background.WithValue(type traceid.ctxKey, val test-1-218549)
}

func ExampleFromContext() {
	// context without set id, should return the default value
	id1 := traceid.FromContext(context.Background(), "default-1-206951")

	fmt.Println(id1)

	// context with set id, should return the existing value
	ctx := traceid.NewContext(context.Background(), "default-2-616841")
	id2 := traceid.FromContext(ctx, "default-3-67890")

	fmt.Println(id2)

	// Output:
	// default-1-206951
	// default-2-616841
}

//nolint:dupword
func ExampleSetHTTPRequestHeaderFromContext() {
	ctx := context.Background()

	// header not set
	r1, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	id1 := traceid.SetHTTPRequestHeaderFromContext(context.Background(), r1, traceid.DefaultHeader, traceid.DefaultValue)

	fmt.Println(id1)
	fmt.Println(r1.Header.Get(traceid.DefaultHeader))

	// header set
	r2, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx = traceid.NewContext(ctx, "test-904117")
	r2 = r2.WithContext(ctx)

	id2 := traceid.SetHTTPRequestHeaderFromContext(ctx, r2, traceid.DefaultHeader, traceid.DefaultValue)

	fmt.Println(id2)
	fmt.Println(r2.Header.Get(traceid.DefaultHeader))

	// Output:
	//
	//
	// test-904117
	// test-904117
}

func ExampleFromHTTPRequestHeader() {
	ctx := context.Background()

	// header not set should return default
	r1, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	v1 := traceid.FromHTTPRequestHeader(r1, traceid.DefaultHeader, "default-1-103993")

	fmt.Println(v1)

	// header set should return actual value
	r2, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	r2.Header.Add(traceid.DefaultHeader, "test-1-413579")

	v2 := traceid.FromHTTPRequestHeader(r2, traceid.DefaultHeader, "default-2-968041")

	fmt.Println(v2)

	// Output:
	// default-1-103993
	// test-1-413579
}
