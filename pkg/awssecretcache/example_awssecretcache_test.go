package awssecretcache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/tecnickcom/nurago/pkg/awssecretcache"
)

// exampleSMClient is a stand-in for the AWS Secrets Manager client,
// implementing only the awssecretcache.SecretsManagerClient interface. Real
// code would let New build the client from aws.Config instead.
type exampleSMClient struct{}

func (exampleSMClient) GetSecretValue(
	_ context.Context,
	params *awssm.GetSecretValueInput,
	_ ...func(*awssm.Options),
) (*awssm.GetSecretValueOutput, error) {
	return &awssm.GetSecretValueOutput{
		SecretString: aws.String("s3cr3t-for-" + aws.ToString(params.SecretId)),
	}, nil
}

func Example() {
	// A caller would normally configure a real client via options such as
	// WithAWSOptions or WithEndpointMutable; here an injected client keeps the
	// example self-contained.
	cache, err := awssecretcache.New(
		context.TODO(),
		128,           // maximum number of cached secrets
		5*time.Minute, // TTL per entry
		awssecretcache.WithSecretsManagerClient(exampleSMClient{}),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// The first call fetches upstream; subsequent calls within the TTL are
	// served from memory, and concurrent callers for the same key share a
	// single in-flight lookup.
	value, err := cache.GetSecretString(context.TODO(), "prod/myapp/db-password")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(value)

	// Output:
	// s3cr3t-for-prod/myapp/db-password
}
