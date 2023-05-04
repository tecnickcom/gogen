FROM gosrvlibexampleowner/dev_gosrvlibexample

ARG HOST_USER="root"
ARG HOST_GROUP="root"

ENV HOST_OWNER=${HOST_USER}:${HOST_GROUP}

RUN mkdir /workspace

# Add only the required project resources
ADD resources /workspace/resources
ADD openapi*.yaml /workspace/
ADD Makefile /workspace
ADD RELEASE /workspace
ADD VERSION /workspace

WORKDIR /workspace
ENTRYPOINT ["/workspace/resources/test/integration/entrypoint.sh"]
