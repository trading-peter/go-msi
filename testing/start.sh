#!/bin/sh
set -ex
unset DOCKER_BUILDKIT
docker build -f testing/Dockerfile -t go-msi-testing:latest testing &&
  docker run --rm -it -v C:/dev/src/github.com/observiq/go-msi:C:/gopath/src/github.com/observiq/go-msi go-msi-testing:latest
