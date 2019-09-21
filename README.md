# gogen

*Command-line tool to generate GO services, applications and libraries with reusable logic.*

[![Master Build Status](https://secure.travis-ci.org/tecnickcom/gogen.png?branch=master)](https://travis-ci.org/tecnickcom/gogen?branch=master)
[![Donate via PayPal](https://img.shields.io/badge/donate-paypal-87ceeb.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&currency_code=GBP&business=paypal@tecnick.com&item_name=donation%20for%20gogen%20project)
*Please consider supporting this project by making a donation via [PayPal](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&currency_code=GBP&business=paypal@tecnick.com&item_name=donation%20for%20gogen%20project)*

* **category**    Tool
* **author**      Nicola Asuni <info@tecnick.com>
* **copyright**   2014-2019 Nicola Asuni - Tecnick.com LTD
* **license**     MIT (see [LICENSE](LICENSE))
* **link**        https://github.com/tecnickcom/gogen


## Description

This is a command-line tool to quickly generate GO services, applications and libraries with a common set of features and reusable logic.

For an equivalent project in Python please check [PyGen](https://github.com/tecnickcom/pygen).

Each GO project built with this tool adheres to the set of conventions detailed in the following articles:

* [Software Naming](https://technick.net/guides/software/software_naming)
* [Software Structure](https://technick.net/guides/software/software_structure)
* [Software Versioning](https://technick.net/guides/software/software_versioning)
* [Software Configuration](https://technick.net/guides/software/software_configuration)
* [Software Logging Format](https://technick.net/guides/software/software_logging_format)
* [Software Metrics](https://technick.net/guides/software/software_metrics)
* [Simple API JSON Response Format](https://technick.net/guides/software/software_json_api_format)
* [Software Automation](https://technick.net/guides/software/software_automation)
* [Build Software with Docker](https://technick.net/guides/software/software_docker_build)

Each generated project is immediately functional and can be fully tested using the ```make qa``` command.

To understand the logic of the generated applications please start with the ```main.go``` file and follow the code.


## Quick Start

This project includes a Makefile that allows you to test and build the project in a Linux-compatible system with simple commands.  
All the artifacts and reports produced using this Makefile are stored in the *target* folder.  

To see all available options:
```
make help
```


## Usage

```
make new TYPE=app CONFIG=myproject.cfg
```

* **TYPE** is the project type:
    * **lib** : Library
    * **app** : Command-line application
    * **srv** : HTTP API service

* **CONFIG** is the configuration file containing the project settings.

To create a new configuration please clone the *default.cfg* file and change the values.

All projects are creted inside the *target* directory and should be moved to the correct path inside the *$GOPATH/src*.


## Features

### Services (srv)

* Web HTTP(S) RESTful JSON API;
* Standard command line options;
* Multiple configuration options, including remote configuration via Consul, Etcd or Environmental variable;
* Logging;
* StatsD client to collect usage metrics;
* Unit tests;
* Makefile;
* Docker build;
* RPM, DEB and Docker packaging;
* Example Proxy endpoint;
* Backend examples in MySQL, MongoDB and ElasticSearch.

### Applications (app)

* Standard command line options;
* Multiple configuration options, including remote configuration via Consul, Etcd or Environmental variable;
* Logging;
* Unit tests;
* Makefile;
* Docker build;
* RPM, DEB and Docker packaging.

### Libraries (lib)

* Unit tests;
* Makefile;
* Docker build;
