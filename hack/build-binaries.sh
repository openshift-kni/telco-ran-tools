#!/bin/bash
set -e
set -o pipefail

HACKDIR=$( dirname $( readlink -f "${BASH_SOURCE[0]}" ) )
BASEDIR="${HACKDIR}/.."

export GO111MODULE=on
export GOPROXY=off
export GOFLAGS=-mod=vendor
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

go build -v -o "${BASEDIR}/_output/factory-precaching-cli" main.go
