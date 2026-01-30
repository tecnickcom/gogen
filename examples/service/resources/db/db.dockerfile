FROM flyway/flyway:12-alpine
ARG DEPLOY_ENV="int"
ARG DBTYPE="mysql"
COPY ./${DBTYPE}/schema/*.sql /flyway/sql/
COPY ./${DBTYPE}/${DEPLOY_ENV}/*sql /flyway/sql/
