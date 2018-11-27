#!/bin/sh
set -ex
unset DOCKER_BUILDKIT
docker build -f testing/Dockerfile -t go-msi-testing:latest testing &&
  docker run --rm -it -v C:/dev/src/github.com/mat007/go-msi:C:/gopath/src/github.com/mat007/go-msi go-msi-testing:latest
