# gogen

*Command-line tool to generate GO applications and libraries*

[![Donate via PayPal](https://img.shields.io/badge/donate-paypal-87ceeb.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&currency_code=GBP&business=paypal@tecnick.com&item_name=donation%20for%20gogen%20project)
*Please consider supporting this project by making a donation via [PayPal](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&currency_code=GBP&business=paypal@tecnick.com&item_name=donation%20for%20gogen%20project)*

* **category**    Tool
* **author**      Nicola Asuni <info@tecnick.com>
* **copyright**   2015-2016 Nicola Asuni - Tecnick.com LTD
* **license**     MIT (see LICENSE)
* **link**        https://github.com/tecnickcom/gogen


## Description

This is a command-line tool to generate GO applications and libraries.

Each GO application and library built with this tool contains a set of common features,
so the developer can immediately start writing the logic code while reusing a common infrastructure.


### Application Features

* Web HTTP RESTful JSON API;
* Standard command line options;
* Multiple configuration options, including remote configuration via Consul or Etcd;
* Logging;
* StatsD client to collect usage metrics;
* Unit tests;

### Library Features

* ...


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

* **TYPE** defines the project type: app or lib
* **CONFIG** specify the configuration file containing the project settings.

To create a new configuration please clone *default.cfg* and change the values.

All projects are creted inside the *target* directory and needs to be moved to the correct path inside the *GOPATH/src*.


## Developer(s) Contact

* Nicola Asuni <info@tecnick.com>
