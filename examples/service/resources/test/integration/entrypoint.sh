#!/usr/bin/env bash

set -e

# wait for resources to be available and run integration tests
dockerize \
    -timeout 30s \
    -wait tcp://gosrvlibexample:8082 \
    echo

# run tests
make openapitest apitest DEPLOY_ENV=int
