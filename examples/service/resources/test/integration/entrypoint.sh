#!/usr/bin/env bash

set -e

# wait for resources to be available and run integration tests
dockerize \
    -timeout 120s \
    -wait tcp://gogenexample:8073/ping \
    -wait http://gogenexample_smocker_ipify:8081/version \
    -wait tcp://gogenexample_mysql:3306 \
    echo

# configure smocker mocks for the ipify client
curl -s -XPOST \
  --header "Content-Type: application/x-yaml" \
  --data-binary "@resources/test/integration/smocker/ipify_apitest.yaml" \
  http://gogenexample_smocker_ipify:8081/mocks

# run tests
DEPLOY_ENV=int make openapitest apitest

# reset the report folder ownership to the host user/group.
chown -R ${HOST_OWNER} /workspace/target/report/
