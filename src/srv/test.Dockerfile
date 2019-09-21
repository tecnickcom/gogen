FROM golang
ENV ~#UPROJECT#~_REMOTECONFIGPROVIDER consul
ENV ~#UPROJECT#~_REMOTECONFIGENDPOINT consul:8500
ENV ~#UPROJECT#~_REMOTECONFIGPATH /config/~#PROJECT#~
WORKDIR /go/src/~#PROJECT#~
COPY . .
RUN echo -e '#!/usr/bin/env bash\nset -e\n'"export ~#UPROJECT#~_REMOTECONFIGPROVIDER=${~#UPROJECT#~_REMOTECONFIGPROVIDER}\nexport ~#UPROJECT#~_REMOTECONFIGENDPOINT=${~#UPROJECT#~_REMOTECONFIGENDPOINT}\nexport ~#UPROJECT#~_REMOTECONFIGPATH=${~#UPROJECT#~_REMOTECONFIGPATH}\nexport ~#UPROJECT#~_REMOTECONFIGSECRETKEYRING=${~#UPROJECT#~_REMOTECONFIGSECRETKEYRING}\n"'sleep 15\ncurl -vvv --request PUT --data @resources/test/etc/~#PROJECT#~/consul.config.json http://consul:8500/v1/kv/config/~#PROJECT#~\ntarget/usr/bin/~#PROJECT#~ -c resources/etc/~#PROJECT#~\nexec "$@"' > /go/src/~#PROJECT#~/dockerstart.sh && chmod +x /go/src/~#PROJECT#~/dockerstart.sh
RUN make build
ENTRYPOINT /go/src/~#PROJECT#~/dockerstart.sh
