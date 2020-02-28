# ~#PROJECT#~

*Brief project description ...*

* **category**    Service
* **copyright**   ~#CURRENTYEAR#~ ~#OWNER#~
* **license**     see [LICENSE](LICENSE)
* **link**         ~#PROJECTLINK#~

-----------------------------------------------------------------

## TOC

* [Description](#description)
* [Requirements](#requirements)
* [Quick Start](#quickstart)
* [Running all tests](#runtest)
* [Usage](#usage)
* [Configuration](#configuration)
* [Examples](#examples)
* [Logs](#logs)
* [Metrics](#metrics)
* [Profiling](#profiling)
* [User Authentication (JWT)](#jwt)
* [OpenApi](#openapi)
* [Docker](#docker)

-----------------------------------------------------------------

<a name="description"></a>
## Description

This service provides a RESTful HTTP(S) JSON API that listen on the configured **address:port**.

-----------------------------------------------------------------

<a name="requirements"></a>
## Requirements

An additional Python program is used to check the validity of the JSON configuration files against a JSON schema:

```
sudo pip install jsonschema
```

-----------------------------------------------------------------

<a name="quickstart"></a>
## Quick Start

This project includes a Makefile that allows you to test and build the project in a Linux-compatible system with simple commands.  
All the artifacts and reports produced using this Makefile are stored in the *target* folder.  

All the packages listed in the *resources/DockerDev/Dockerfile* file are required in order to build and test all the library options in the current environment. Alternatively, everything can be built inside a [Docker](https://www.docker.com) container using the command "make dbuild".

To see all available options:
```
make help
```

To build the project inside a Docker container (requires Docker):
```
make dbuild
```

An arbitrary make target can be executed inside a Docker container by specifying the "MAKETARGET" parameter:
```
MAKETARGET='qa' make dbuild
```
The list of make targets can be obtained by typing ```make```


The base Docker building environment is defined in the following Dockerfile:
```
resources/DockerDev/Dockerfile
```

To execute all the default test builds and generate reports in the current environment:
```
make qa
```

To format the code (please use this command before submitting any pull request):
```
make format
```

-----------------------------------------------------------------

<a name="runtest"></a>
## Running all tests

Before committing the code, please check if it passes all tests using
```bash
make qa
```

Other make options are available install this library globally and build RPM and DEB packages.
Please check all the available options using `make help`.

-----------------------------------------------------------------

<a name="usage"></a>
## Usage

```bash
~#PROJECT#~ [flags]

Flags:

-c, --configDir  string  Configuration directory to be added on top of the search list
-o, --loglevel   string  Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
```

----------------------------------------------------------------

<a name="configuration"></a>
## Configuration

See [CONFIG.md](CONFIG.md).

-----------------------------------------------------------------

<a name="examples"></a>
## Examples

Once the application has being compiled with `make build`, it can be quickly tested:

```bash
target/usr/bin/~#PROJECT#~ -c resources/test/etc/~#PROJECT#~
```

<a name="logs"></a>
## Logs

This program logs the log messages in JSON format:

```
{
    "URI":"/",
    "code":200,
    "datetime":"2020-10-06T14:56:48Z",
    "hostname":"myserver",
    "level":"info",
    "msg":"request",
    "program":"~#PROJECT#~",
    "release":"1",
    "timestamp":1475765808084372773,
    "type":"GET",
    "version":"1.0.0"
}
```

Logs are sent to stderr by default.

The log level can be set either in the configuration or as command argument (`logLevel`).

-----------------------------------------------------------------

<a name="metrics"></a>
## Metrics

This service provides [Prometheus](https://prometheus.io/) metrics at the `/metrics` endpoint.

[Grafana](https://grafana.com/) dashboards are available at `resources/grafana/`.

-----------------------------------------------------------------

<a name="profiling"></a>
## Profiling

This service provides [PPROF](https://github.com/google/pprof) profiling data at the `/pprof` endpoint.

The pprof data can be analyzed and displayed using the pprof tool:

```
go get github.com/google/pprof
```

Example:

```
pprof -seconds 10 -http=localhost:8182 http://INSTANCE_URL:PORT/pprof/profile
```

-----------------------------------------------------------------

<a name="jwt"></a>
## User Authentication (JWT)

This service includes optional support for JWT authentication.
The configuration file contains user accounts that are allowed to access the restricted API endpoints.
The password field contains the hash version of the original password generated using the *hash* tool in resources/hash.
```
./resources/hash/hash 'jwttest'
```
The default password in the test file is `jwttest`.

To get the JWT authentication token send a POST request with the user credentials.
For example:

```
curl -k -d '{"username":"test","password":"jwttest"}' -H "Content-Type: application/json" -X POST https://localhost:8017/auth/login

```

this returns the JWT token in a JSON data field:

```
{
  "program": "~#PROJECT#~",
  "version": "1.0.0",
  "release": "1",
  "url": ":8017",
  "datetime": "2020-11-21T17:17:29Z",
  "timestamp": 1574356649975251500,
  "status": "success",
  "code": 200,
  "message": "OK",
  "data": "abCDbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQzNTY5NDl9.z51YUiSkDEE78TOORLcjF5fkeIEG6jT_E64luKlogEw"
}
```

To get JWT the token with *curl* and *jq*:

```
TOKEN=$(curl -s -k -d '{"username":"test","password":"jwttest"}' -H "Content-Type: application/json" -X POST https://localhost:8017/auth/login | jq -r .data)
```
```
echo $TOKEN
abCDbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQzNTcwNjV9.imrC22sivbTLVgsaSDIL_GG9N6FDOkhl0S_BNobWxus
```

The secured endpoints can be accessed by specifying the authorization token:

```
curl -s -k -H "Authorization: Bearer $TOKEN" https://localhost:8017/db/status
```

Before the expiration the token can be renewed using the auth/refresh endpoint:

```
TOKEN=$(curl -s -k -H "Authorization: Bearer $TOKEN" https://localhost:8017/auth/refresh | jq -r .data)
```
```
echo $TOKEN
abCDbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQ0Mzk3ODV9.uw7PIX2Pjqh_UJd1jTQ7JN6bRNGXmSB4ThHra0kfqBg
```

-----------------------------------------------------------------

<a name="openapi"></a>
## OpenAPI

The ~#PROJECT#~ API is specified via the [OpenAPI 3](https://www.openapis.org/) file: `openapi.yaml`.

The openapi file can be edited using the Swagger Editor:

```
docker pull swaggerapi/swagger-editor
docker run -p 8056:8080 swaggerapi/swagger-editor
```

and pointing the Web browser to http://localhost:8056


### API Testing

The live API can be tested against the OpenAPI specification file by installing [schemathesis](https://github.com/kiwicom/schemathesis) and run the command:

```
schemathesis run --validate-schema=false --checks=all --base-url=http://~#PROJECT#~.qa:8017 openapi.yaml
```

or

```
~#UPROJECT#~_URL=http://~#PROJECT#~.qa:8017 make openapitest
```

-----------------------------------------------------------------

<a name="docker"></a>
## Docker

To build a Docker scratch container for the ~#PROJECT#~ executable binary execute the following command:
```
make docker
```

To push the Docker container in our ECR repo execute:
```
make dockerpush
```
Note that this command will require to set the follwoing environmental variables or having an AWS profile installed:

* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`
* `AWS_DEFAULT_REGION`


### Useful Docker commands

To manually create the container you can execute:
```
docker build --tag="~#OWNER#~/~#PROJECT#~dev" .
```

To log into the newly created container:
```
docker run -t -i ~#OWNER#~/~#PROJECT#~dev /bin/bash
```

To get the container ID:
```
CONTAINER_ID=`docker ps -a | grep ~#OWNER#~/~#PROJECT#~dev | cut -c1-12`
```

To delete the newly created docker container:
```
docker rm -f $CONTAINER_ID
```

To delete the docker image:
```
docker rmi -f ~#OWNER#~/~#PROJECT#~dev
```

To delete all containers
```
docker rm $(docker ps -a -q)
```

To delete all images
```
docker rmi $(docker images -q)
```
