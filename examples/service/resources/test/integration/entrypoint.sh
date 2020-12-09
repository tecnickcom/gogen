#!/usr/bin/env bash

set -e

# wait for resources to be available and run integration tests
dockerize \
    -timeout 30s \
    -wait tcp://gosrvlibexample:8072/ping \
    -wait http://gosrvlibexample_smocker_ipify:8081/version \
    echo

# configure smocker mocks for the ipify client
curl -s -XPOST \
  --header "Content-Type: application/x-yaml" \
  --data-binary "@resources/test/integration/smocker/ipify_apitest.yaml" \
  http://gosrvlibexample_smocker_ipify:8081/mocks

# run tests
make openapitest apitest DEPLOY_ENV=int
