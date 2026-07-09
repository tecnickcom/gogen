package s3_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tecnickcom/gogen/pkg/s3"
)

// exampleS3Client is a minimal in-memory stand-in for the AWS S3 client,
// implementing only the s3.S3 interface. Real code would let New build the
// client from aws.Config instead.
type exampleS3Client struct {
	objects map[string][]byte
}

func (c *exampleS3Client) PutObject(
	_ context.Context,
	params *awss3.PutObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.PutObjectOutput, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read object body: %w", err)
	}

	c.objects[aws.ToString(params.Key)] = body

	return &awss3.PutObjectOutput{}, nil
}

func (c *exampleS3Client) GetObject(
	_ context.Context,
	params *awss3.GetObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.GetObjectOutput, error) {
	body := c.objects[aws.ToString(params.Key)]

	return &awss3.GetObjectOutput{
		Body:          io.NopCloser(strings.NewReader(string(body))),
		ContentType:   aws.String("application/json"),
		ContentLength: aws.Int64(int64(len(body))),
	}, nil
}

func (c *exampleS3Client) ListObjectsV2(
	_ context.Context,
	_ *awss3.ListObjectsV2Input,
	_ ...func(*awss3.Options),
) (*awss3.ListObjectsV2Output, error) {
	contents := make([]types.Object, 0, len(c.objects))
	for key := range c.objects {
		contents = append(contents, types.Object{Key: aws.String(key)})
	}

	return &awss3.ListObjectsV2Output{Contents: contents}, nil
}

func (c *exampleS3Client) DeleteObject(
	_ context.Context,
	params *awss3.DeleteObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.DeleteObjectOutput, error) {
	delete(c.objects, aws.ToString(params.Key))

	return &awss3.DeleteObjectOutput{}, nil
}

func (c *exampleS3Client) HeadBucket(
	_ context.Context,
	_ *awss3.HeadBucketInput,
	_ ...func(*awss3.Options),
) (*awss3.HeadBucketOutput, error) {
	return &awss3.HeadBucketOutput{}, nil
}

func Example() {
	// A caller would normally configure a real client via options such as
	// WithAWSOptions or WithEndpointMutable; here an injected client keeps the
	// example self-contained, so no AWS configuration is loaded.
	c, err := s3.New(
		context.TODO(),
		"my-bucket",
		s3.WithS3Client(&exampleS3Client{objects: map[string][]byte{}}),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	err = c.Put(context.TODO(), "reports/latest.json", strings.NewReader(`{"ok":true}`))
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	obj, err := c.Get(context.TODO(), "reports/latest.json")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	data, err := io.ReadAll(obj.Body())
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	err = obj.Close()
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(obj.ContentType())
	fmt.Println(obj.ContentLength())
	fmt.Println(string(data))

	keys, err := c.ListKeys(context.TODO(), "reports/")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(keys)

	err = c.Delete(context.TODO(), "reports/latest.json")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// Output:
	// application/json
	// 11
	// {"ok":true}
	// [reports/latest.json]
}
