#!/usr/bin/env bash
set -e
export ~#UPROJECT#~_REMOTECONFIGPROVIDER=
export ~#UPROJECT#~_REMOTECONFIGENDPOINT=
export ~#UPROJECT#~_REMOTECONFIGPATH=
export ~#UPROJECT#~_REMOTECONFIGSECRETKEYRING=test
curl -vvv --request PUT --data @resources/test/etc/~#PROJECT#~/consul.config.json http://consul:8500/v1/kv/config/~#PROJECT#~
target/usr/bin/~#PROJECT#~ -c resources/etc/~#PROJECT#~
exec "$@"
