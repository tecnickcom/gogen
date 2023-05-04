#!/usr/bin/env bash
#
# dockerbuild.sh
#
# Build the software inside a Docker container
#
# @author      Nicola Asuni <info@tecnick.com>
# ------------------------------------------------------------------------------
set -e -u +x

# NOTES:
# This script requires Docker

# EXAMPLE USAGE:
# CVSPATH=project VENDOR=vendorname PROJECT=projectname MAKETARGET=buildall ./dockerbuild.sh

# Variables (parameters).
: ${CVSPATH:=project}
: ${VENDOR:=vendor}
: ${PROJECT:=project}
: ${MAKETARGET:=format clean mod deps gendoc generate qa build}
: ${SSH_PRIVATE_KEY:=$(cat ~/.ssh/id_rsa || cat ~/.ssh/id_ed25519)}
: ${SSH_PUBLIC_KEY:=$(cat ~/.ssh/id_rsa.pub || cat ~/.ssh/id_ed25519.pub)}
: ${DOCKER:=$(which docker)}
: ${DOCKERDEV:=${VENDOR}/dev_${PROJECT}}

# Build the base environment and keep it cached locally.
${DOCKER} build --pull --tag ${DOCKERDEV} --file ./resources/docker/Dockerfile.dev ./resources/docker/

# Define the project root path.
PRJPATH=/root/src/${CVSPATH}/${PROJECT}

# Generate a temporary Dockerfile to build and test the project
# NOTE: The exit status of the RUN command is stored to be returned later,
#       so in case of error we can continue without interrupting this script.
cat > Dockerfile.test <<- EOM
FROM ${DOCKERDEV}
ARG SSH_PRIVATE_KEY=""
ARG SSH_PUBLIC_KEY=""
RUN \\
mkdir -p /root/.ssh \\
&& echo "\${SSH_PRIVATE_KEY}" > /root/.ssh/id_rsa \\
&& echo "\${SSH_PUBLIC_KEY}" > /root/.ssh/id_rsa.pub \\
&& echo "Host *" >> /root/.ssh/config \\
&& echo "    StrictHostKeyChecking no" >> /root/.ssh/config \\
&& echo "    GlobalKnownHostsFile  /dev/null" >> /root/.ssh/config \\
&& echo "    UserKnownHostsFile    /dev/null" >> /root/.ssh/config \\
&& chmod 600 /root/.ssh/id_rsa \\
&& chmod 644 /root/.ssh/id_rsa.pub \\
&& echo "[user]" >> /root/.gitconfig \\
&& echo "	email = godev@example.com" >> /root/.gitconfig \\
&& echo "	name = godevlocaltestuser" >> /root/.gitconfig \\
&& echo "[url \"ssh://git@${CVSPATH}\"]" >> /root/.gitconfig \\
&& echo "	insteadOf = https://${CVSPATH}" >> /root/.gitconfig \\
&& mkdir -p ${PRJPATH}
ADD ./ ${PRJPATH}
WORKDIR ${PRJPATH}
RUN make ${MAKETARGET} || (echo \$? > target/make.exit)
EOM

# Define the temporary Docker image name.
DOCKER_IMAGE_NAME=${VENDOR}/build_${PROJECT}

# Build the Docker image.
BUILDKIT_PROGRESS=plain \
${DOCKER} build \
--no-cache \
--build-arg SSH_PRIVATE_KEY="${SSH_PRIVATE_KEY}" \
--build-arg SSH_PUBLIC_KEY="${SSH_PUBLIC_KEY}" \
--tag ${DOCKER_IMAGE_NAME} \
--file Dockerfile.test .

# Start a container using the newly created Docker image.
CONTAINER_ID=$(docker run -d ${DOCKER_IMAGE_NAME})

# Copy all build/test artifacts back to the host.
${DOCKER} cp ${CONTAINER_ID}:"${PRJPATH}/target" ./

# Remove the temporary container and image.
rm -f Dockerfile.test
${DOCKER} rm -f ${CONTAINER_ID} || true
${DOCKER} rmi -f ${DOCKER_IMAGE_NAME} || true
