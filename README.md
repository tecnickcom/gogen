<!-- Space: APIOSS -->
<!-- Parent: Projects -->
<!-- Title: gosrvlib -->

# gosrvlib

*Go Service Library*

This Open Source project contains a collection of common GO packages that forms the base structure of a service.

[![check](https://github.com/nexmoinc/gosrvlib/actions/workflows/check.yaml/badge.svg)](https://github.com/nexmoinc/gosrvlib/actions/workflows/check.yaml)
[![whitesource](https://github.com/nexmoinc/gosrvlib/actions/workflows/whitesource.yaml/badge.svg)](https://github.com/nexmoinc/gosrvlib/actions/workflows/whitesource.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nexmoinc/gosrvlib)](https://goreportcard.com/report/github.com/nexmoinc/gosrvlib)
[![Go Reference](https://pkg.go.dev/badge/github.com/nexmoinc/gosrvlib.svg)](https://pkg.go.dev/github.com/nexmoinc/gosrvlib)


* **category**    Library
* **license**     [MIT](https://github.com/nexmoinc/gosrvlib/blob/main/LICENSE)
* **link**        https://github.com/nexmoinc/gosrvlib

-----------------------------------------------------------------

## TOC

* [Quick Start](#quickstart)
* [Running all tests](#runtest)
* [Examples](#examples)

-----------------------------------------------------------------

<a name="quickstart"></a>
## Quick Start

This project includes a Makefile that allows you to test and build the project in a Linux-compatible system with simple commands.  
All the artifacts and reports produced using this Makefile are stored in the *target* folder.  

All the packages listed in the *resources/docker/Dockerfile* file are required in order to build and test all the library options in the current environment.
Alternatively, everything can be built inside a [Docker](https://www.docker.com) container using the command "make dbuild".

To see all available options:
```bash
make help
```

To build the project inside a Docker container (requires Docker):
```bash
make dbuild
```

An arbitrary make target can be executed inside a Docker container by specifying the "MAKETARGET" parameter:
```bash
MAKETARGET='deps mod qa example' make dbuild
```
The list of make targets can be obtained by typing ```make```


The base Docker building environment is defined in the following Dockerfile:
```bash
resources/docker/Dockerfile.dev
```

To download all dependencies:
```bash
make deps
```

To update the mod file:
```bash
make mod
```

To execute all the default test builds and generate reports in the current environment:
```bash
make qa
```

To format the code (please use this command before submitting any pull request):
```bash
make format
```

-----------------------------------------------------------------

<a name="runtest"></a>
## Running all tests

Before committing the code, please format it and check if it passes all tests using
```bash
DEVMODE=LOCAL make format clean mod deps generate qa example
```

-----------------------------------------------------------------

<a name="examples"></a>
## Examples

Please check the `examples` folder for an example of a service based on this library.

The following command generates a new project from the example using the data set in the project.cfg file:

```bash
make project CONFIG=project.cfg
```
