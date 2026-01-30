FROM flyway/flyway:12-alpine
ARG DEPLOY_ENV="int"
ARG DBTYPE="mysql"
COPY ./${DBTYPE}/V000__schema.sql /flyway/sql/V000__schema.sql
COPY ./${DBTYPE}/${DEPLOY_ENV}/*sql /flyway/sql/
