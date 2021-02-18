FROM golang:1.16

RUN mkdir /workspace

# Schemathesis
RUN apt update \
    && apt -y upgrade \
    && apt install -y python3-pip \
    && pip3 install schemathesis

# Venom
ADD https://github.com/ovh/venom/releases/download/v0.28.0/venom.linux-amd64 /usr/bin/venom
RUN chmod ug+x /usr/bin/venom

# Add only the required project resources
ADD resources /workspace/resources
ADD openapi*.yaml /workspace/
ADD Makefile /workspace
ADD RELEASE /workspace
ADD VERSION /workspace

WORKDIR /workspace
ENTRYPOINT ["/workspace/resources/test/integration/entrypoint.sh"]
