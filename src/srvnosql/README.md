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
sudo pip install json-spec 
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

If no command-line parameters are specified, then the ones in the configuration file (**config.json**) will be used.
The configuration files can be stored in the current directory or in any of the following (in order of precedence):
* ./
* config/
* $HOME/~#PROJECT#~/
* /etc/~#PROJECT#~/

This program also support secure remote configuration via Consul or Etcd.
The remote configuration server can be defined either in the local configuration file using the following parameters, or with environment variables:

* **remoteConfigProvider** : remote configuration source ("consul", "etcd");
* **remoteConfigEndpoint** : remote configuration URL (ip:port);
* **remoteConfigPath** : remote configuration path where to search fo the configuration file (e.g. "/config/~#PROJECT#~");
* **remoteConfigSecretKeyring** : path to the openpgp secret keyring used to decript the remote configuration data (e.g. "/etc/~#PROJECT#~/configkey.gpg"); if empty a non secure connection will be used instead;

The equivalent environment variables are:

* ~#UPROJECT#~_REMOTECONFIGPROVIDER
* ~#UPROJECT#~_REMOTECONFIGENDPOINT
* ~#UPROJECT#~_REMOTECONFIGPATH
* ~#UPROJECT#~_REMOTECONFIGSECRETKEYRING


## Server Mode

When the server mode is enabled a RESTful HTTP JSON API server will listen on the configured **address:port** for the following entry points:

| ENTRY POINT                   | METHOD | DESCRIPTION                                                    |
|:----------------------------- |:------:|:-------------------------------------------------------------- |
|<nobr> /                </nobr>| GET    |<nobr> return a list of available entry points and tests </nobr>|
|<nobr> /status          </nobr>| GET    |<nobr> check the server status                           </nobr>|


## Examples

Once the application has being compiled with `make build`, it can be quickly tested:

```bash
target/usr/bin/~#PROJECT#~
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
    "version":"3.4.0"
}
```

