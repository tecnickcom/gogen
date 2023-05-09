# Configuration Guide

The gosrvlibexample service can load the configuration either from a local configuration file or remotely via [Consul](https://www.consul.io/), [Etcd](https://github.com/coreos/etcd) or a single Environmental Variable.

The local configuration file is always loaded before the remote configuration, the latter always overwrites any local setting.

If the *configDir* parameter is not specified, then the program searches for a **config.json** file in the following directories (in order of precedence):

* ./
* $HOME/gosrvlibexample/
* /etc/gosrvlibexample/


## Default Configuration

The default configuration file is installed in the **/etc/gosrvlibexample/** folder (**config.json**) along with the JSON schema **config.schema.json**.


## Remote Configuration

This program supports secure remote configuration via Consul, Etcd or single environment variable.
The remote configuration server can be defined either in the local configuration file using the following parameters, or with environment variables:

The configuration fields are:

* **remoteConfigProvider**      : Remote configuration source ("consul", "etcd", "envvar")
* **remoteConfigEndpoint**      : Remote configuration URL (ip:port)
* **remoteConfigPath**          : Remote configuration path in which to search for the configuration file (e.g. "/config/gosrvlibexample")
* **remoteConfigSecretKeyring** : Path to the [OpenPGP](http://openpgp.org/) secret keyring used to decrypt the remote configuration data (e.g. "/etc/gosrvlibexample/configkey.gpg"); if empty a non secure connection will be used instead
* **remoteConfigData**          : Base64 encoded JSON configuration data to be used with the "envvar" provider

The equivalent environment variables are:

* GOSRVLIBEXAMPLE_REMOTECONFIGPROVIDER
* GOSRVLIBEXAMPLE_REMOTECONFIGENDPOINT
* GOSRVLIBEXAMPLE_REMOTECONFIGPATH
* GOSRVLIBEXAMPLE_REMOTECONFIGSECRETKEYRING
* GOSRVLIBEXAMPLE_REMOTECONFIGDATA


## Configuration Format

The configuration format is a single JSON structure with the following fields:

* **remoteConfigProvider**      : Remote configuration source ("consul", "etcd", "envvar")
* **remoteConfigEndpoint**      : Remote configuration URL (ip:port)
* **remoteConfigPath**          : Remote configuration path in which to search for the configuration file (e.g. "/config/gosrvlibexample")
* **remoteConfigSecretKeyring** : Path to the openpgp secret keyring used to decrypt the remote configuration data (e.g. "/etc/gosrvlibexample/configkey.gpg"); if empty a non secure connection will be used instead

* **enabled**: Enable or disable the service

* **log**:  Logging settings
    * **format**:  Logging format: CONSOLE, JSON
    * **level**:   Defines the default log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
    * **network**: (OPTIONAL) Network type used by the Syslog (i.e. udp or tcp)
    * **address**: (OPTIONAL) Network address of the Syslog daemon (ip:port) or just (:port)

* **shutdown_timeout**: Time to wait on exit for a graceful shutdown [seconds]

* **servers**: Configuration for exposed servers
    * **monitoring**: Monitoring HTTP server
        * **address**: HTTP address (ip:port) or just (:port)
        * **timeout**: HTTP request timeout [seconds]
    * **public**: *Public HTTP server*
        * **address**: HTTP address (ip:port) or just (:port)
        * **timeout**: HTTP request timeout [seconds]

* **clients**: Configuration for external service clients
    * **ipify**:  ipify service client
        * **address**:  Base URL of the service
        * **timeout**:  HTTP client timeout [seconds]


## Formatting Configuration

All configuration files are formatted and ordered by key using the [jq](https://github.com/stedolan/jq) tool.
For example:

```cat 'resources/etc/gosrvlibexample/config.schema.json' | jq -S .```


## Validating Configuration

The [check-jsonschema](https://github.com/python-jsonschema/check-jsonschema) Python program can be used to check the validity of the configuration file against the JSON schema.
It can be installed using the Python pip install tool:

```
sudo pip install check-jsonschema
```

Example usage:

```
check-jsonschema --schemafile resources/etc/gosrvlibexample/config.schema.json resources/etc/gosrvlibexample/config.json
```
