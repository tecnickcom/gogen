/*
Package s3 provides a lightweight wrapper around the AWS SDK v2 S3 client for
common bucket object operations.

The package focuses on pragmatic day-to-day workflows:

  - upload object data,
  - download object data,
  - list object keys by prefix,
  - delete objects.

It is built on github.com/aws/aws-sdk-go-v2/service/s3 while hiding repetitive
client setup and request boilerplate behind a compact API.

# Problem

Using the raw S3 SDK directly in every service often leads to duplicated code
for configuration loading, client initialization, endpoint overrides (for local
testing), and basic object operations. Over time, these duplicated snippets can
drift and make maintenance harder.

This package centralizes that integration into a small reusable client.

# What It Provides

  - [New] to create a bucket-scoped [Client].
  - [Client.Put] to upload from an [io.Reader].
  - [Client.Get] to fetch an object and access its body stream.
  - [Client.ListKeys] to list object keys, optionally filtered by prefix.
  - [Client.Delete] to remove an object by key.

# Configuration & Extensibility

The client configuration composes with [github.com/tecnickcom/gogen/pkg/awsopt]
and exposes option hooks for advanced scenarios:

  - [WithAWSOptions] to pass generic AWS config options,
  - [WithSrvOptionFuncs] to customize S3 service options,
  - [WithEndpointMutable] and [WithEndpointImmutable] for endpoint overrides
    (useful for local S3-compatible environments and tests).

# Benefits

  - Consistent S3 integration patterns across services.
  - Less boilerplate for the most common object operations.
  - Easier local testing and custom endpoint routing.

# Usage

	c, err := s3.New(ctx, "my-bucket")
	if err != nil {
	    return err
	}

	if err := c.Put(ctx, "reports/latest.json", reader); err != nil {
	    return err
	}

	obj, err := c.Get(ctx, "reports/latest.json")
	if err != nil {
	    return err
	}
	_ = obj

	keys, err := c.ListKeys(ctx, "reports/")
	if err != nil {
	    return err
	}
	_ = keys

	if err := c.Delete(ctx, "reports/old.json"); err != nil {
	    return err
	}
*/
package s3
