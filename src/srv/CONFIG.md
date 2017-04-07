# Configuration Guide

The ~#PROJECT#~ service can load the configuration either from a local configuration file or remotely via [Consul](https://www.consul.io/) or [Etcd](https://github.com/coreos/etcd).

The local configuration file is always loaded before the remote configuration, the latter always overwrites any local setting.

If the *configDir* parameter is not specified, then the program searches for a **config.json** file in the following directories (in order of precedence):
* ./
* config/
* $HOME/~#PROJECT#~/
* /etc/~#PROJECT#~/


## Default Configuration

The default configuration file is installed in the **/etc/~#PROJECT#~/** folder along with the example configuration file **config.example.json** and the JSON schema **config.schema.json**.
The example configuration file contains multiple *Service Providers* definitions but all sensitive data has been removed or replaced with "******".


## Remote Configuration

The remote configuration endpoint can be configured either in the local config file or by setting some environmental variables.
The use of environmental variables is particularly important when the program is running inside a Docker container.

The configuration fields are:

* **remoteConfigProvider** : remote configuration source ("consul", "etcd")
* **remoteConfigEndpoint** : remote configuration URL (ip:port)
* **remoteConfigPath** : remote configuration path in which to search for the configuration file (e.g. "/config/~#PROJECT#~")
* **remoteConfigSecretKeyring** : path to the [OpenPGP](http://openpgp.org/) secret keyring used to decrypt the remote configuration data (e.g. "/etc/~#PROJECT#~/configkey.gpg"); if empty a non secure connection will be used instead

The equivalent environment variables are:

* ~#UPROJECT#~_REMOTECONFIGPROVIDER
* ~#UPROJECT#~_REMOTECONFIGENDPOINT
* ~#UPROJECT#~_REMOTECONFIGPATH
* ~#UPROJECT#~_REMOTECONFIGSECRETKEYRING


## Configuration Format

The configuration format is a single JSON structure with the following fields:


* **remoteConfigProvider** :      Remote configuration source ("consul", "etcd")
* **remoteConfigEndpoint** :      Remote configuration URL (ip:port)
* **remoteConfigPath** :          Remote configuration path in which to search for the configuration file (e.g. "/config/~#PROJECT#~")
* **remoteConfigSecretKeyring** : Path to the openpgp secret keyring used to decrypt the remote configuration data (e.g. "/etc/~#PROJECT#~/configkey.gpg"); if empty a non secure connection will be used instead

* **log**:  *Logging settings*
    * **level**:   Defines the default log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
    * **network**: (OPTIONAL) Network type used by the Syslog (i.e. udp or tcp)
    * **address**: (OPTIONAL) Network address of the Syslog daemon (ip:port) or just (:port)

* **stats**:  *StatsD is used to collect usage metrics*
    * **prefix**:       StatsD client string prefix that will be used in every bucket name
    * **network**:      Network type used by the StatsD client (i.e. udp or tcp)
    * **address**:      Network address of the StatsD daemon (ip:port) or just (:port)
    * **flush_period**: Sets how often (in milliseconds) the StatsD client's buffer is flushed. When 0 the buffer is only flushed when it is full

* **serverAddress**:              Internal HTTP address (ip:port) or just (:port)

## Validate Configuration

The json-spec Python program can be used to check the validity of the configuration file against the JSON schema.
It can be installed using the Python pip install tool:

```
sudo pip install json-spec 
```

Example usage:

```
json validate --schema-file=/etc/~#PROJECT#~/config.schema.json --document-file=/etc/~#PROJECT#~/config.json
```
