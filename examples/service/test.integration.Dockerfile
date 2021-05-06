FROM gosrvlibexample/dev_gosrvlibexample

RUN mkdir /workspace

# Add only the required project resources
ADD resources /workspace/resources
ADD openapi*.yaml /workspace/
ADD Makefile /workspace
ADD RELEASE /workspace
ADD VERSION /workspace

WORKDIR /workspace
ENTRYPOINT ["/workspace/resources/test/integration/entrypoint.sh"]
