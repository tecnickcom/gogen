# ~#PROJECT#~

*Brief project description ...*

* **category**    Application
* **copyright**   ~#CURRENTYEAR#~ ~#OWNER#~
* **license**     see [LICENSE](LICENSE)
* **link**        ~#PROJECTLINK#~


## Description

Full project description ...


## Requirements

An additional Python program is used to check the validity of the JSON configuration files against a JSON schema:

```
sudo pip install jsonschema
```

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

## Useful Docker commands

To manually create the container you can execute:
```
docker build --tag="~#VENDOR#~/~#PROJECT#~dev" .
```

To log into the newly created container:
```
docker run -t -i ~#VENDOR#~/~#PROJECT#~dev /bin/bash
```

To get the container ID:
```
CONTAINER_ID=`docker ps -a | grep ~#VENDOR#~/~#PROJECT#~dev | cut -c1-12`
```

To delete the newly created docker container:
```
docker rm -f $CONTAINER_ID
```

To delete the docker image:
```
docker rmi -f ~#VENDOR#~/~#PROJECT#~dev
```

To delete all containers
```
docker rm $(docker ps -a -q)
```

To delete all images
```
docker rmi $(docker images -q)
```

## Running all tests

Before committing the code, please check if it passes all tests using
```bash
make qa
```

Other make options are available install this library globally and build RPM and DEB packages.
Please check all the available options using `make help`.


## Usage

```bash
~#PROJECT#~ [flags]

Flags:

-c, --configDir  string  Configuration directory to be added on top of the search list
-o, --loglevel   string  Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
```

## Configuration

See [CONFIG.md](CONFIG.md).


## Examples

Once the application has being compiled with `make build`, it can be quickly tested:

```bash
target/usr/bin/~#PROJECT#~ -c ../../../resources/test/etc/~#PROJECT#~
```

## Logs

This program logs the log messages in JSON format:

```
{
    "URI":"/",
    "code":200,
    "datetime":"2016-10-06T14:56:48Z",
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


## TLS (HTTPS)

To convert PEM file format to JSON for the configuration file:
```
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' cert.pem
```

## User Authentication

The configuration file contains user accounts that are allowed to access the API.
The password field contains the hash version of the original password generated using the *hash* tool in resources/hash.
```
./resources/hash/hash 'jwttest'

$2a$04$GfYChjSytr0zgLYbSJoyK.XZGbiNm4F5VY08WL0bHBAKgnq3AkcZu
```
The default password in the test file is "jwttest".

To get the JWT authentication token  send a POST request with the user credentials.
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
  "datetime": "2016-06-09T15:11:12Z",
  "timestamp": 1574356649975251500,
  "status": "success",
  "code": 200,
  "message": "OK",
  "data": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQzNTY5NDl9.z51YUiSkDEE78TOORLcjF5fkeIEG6jT_E64luKlogEw"
}
```

To get JWT the token with *curl* and *jq*:

```
TOKEN=$(curl -s -k -d '{"username":"test","password":"jwttest"}' -H "Content-Type: application/json" -X POST https://localhost:8017/auth/login | jq -r .data)
```
```
echo $TOKEN
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQzNTcwNjV9.imrC22sivbTLVgsaSDIL_GG9N6FDOkhl0S_BNobWxus
```

The secured endpoints can be accessed by specifying the authorization token:

```
curl -s -k -H "Authorization: Bearer $TOKEN" https://localhost:8017/status
```

Before the expiration the token can be renewed using the auth/refresh endpoint:

```
TOKEN=$(curl -s -k -H "Authorization: Bearer $TOKEN" https://localhost:8017/auth/refresh | jq -r .data)
```
```
echo $TOKEN
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQ0Mzk3ODV9.uw7PIX2Pjqh_UJd1jTQ7JN6bRNGXmSB4ThHra0kfqBg
```


