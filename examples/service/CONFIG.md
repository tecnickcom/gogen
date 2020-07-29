# Configuration Guide

The srvxmplname service can load the configuration either from a local configuration file or remotely via [Consul](https://www.consul.io/), [Etcd](https://github.com/coreos/etcd) or a single Environmental Variable.

The local configuration file is always loaded before the remote configuration, the latter always overwrites any local setting.

If the *configDir* parameter is not specified, then the program searches for a **config.json** file in the following directories (in order of precedence):
* ./
* config/
* $HOME/srvxmplname/
* /etc/srvxmplname/


## Default Configuration

The default configuration file is installed in the **/etc/srvxmplname/** folder (**config.json**) along with the JSON schema **config.schema.json**.


## Remote Configuration

This program also support secure remote configuration via Consul, Etcd or single environment variable.
The remote configuration server can be defined either in the local configuration file using the following parameters, or with environment variables:

The configuration fields are:

* **remoteConfigProvider**      : remote configuration source ("consul", "etcd", "envvar");
* **remoteConfigEndpoint**      : remote configuration URL (ip:port);
* **remoteConfigPath**          : remote configuration path in which to search for the configuration file (e.g. "/config/srvxmplname");
* **remoteConfigSecretKeyring** : path to the [OpenPGP](http://openpgp.org/) secret keyring used to decrypt the remote configuration data (e.g. "/etc/srvxmplname/configkey.gpg"); if empty a non secure connection will be used instead;
* **remoteConfigData**          : base64 encoded JSON configuration data to be used with the "envvar" provider.

The equivalent environment variables are:

* SRVXMPLENVPREFIX_REMOTECONFIGPROVIDER
* SRVXMPLENVPREFIX_REMOTECONFIGENDPOINT
* SRVXMPLENVPREFIX_REMOTECONFIGPATH
* SRVXMPLENVPREFIX_REMOTECONFIGSECRETKEYRING
* SRVXMPLENVPREFIX_REMOTECONFIGDATA


## Configuration Format

The configuration format is a single JSON structure with the following fields:

* **remoteConfigProvider**      : Remote configuration source ("consul", "etcd", "envvar")
* **remoteConfigEndpoint**      : Remote configuration URL (ip:port)
* **remoteConfigPath**          : Remote configuration path in which to search for the configuration file (e.g. "/config/srvxmplname")
* **remoteConfigSecretKeyring** : Path to the openpgp secret keyring used to decrypt the remote configuration data (e.g. "/etc/srvxmplname/configkey.gpg"); if empty a non secure connection will be used instead

* **enabled**: Enable or disable the service

* **monitoring_address**: Monitoring HTTP address (ip:port) or just (:port)
* **server_address**: Service HTTP address (ip:port) or just (:port)

* **log**:  *Logging settings*
    * **format**:  Logging format: CONSOLE, JSON
    * **level**:   Defines the default log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
    * **network**: (OPTIONAL) Network type used by the Syslog (i.e. udp or tcp)
    * **address**: (OPTIONAL) Network address of the Syslog daemon (ip:port) or just (:port)


## Validate Configuration

The jsonschema Python program can be used to check the validity of the configuration file against the JSON schema.
It can be installed using the Python pip install tool:

```
sudo pip install jsonschema
```

Example usage:

```
json validate --schema-file=/etc/srvxmplname/config.schema.json --document-file=/etc/srvxmplname/config.json
```
