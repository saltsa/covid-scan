#!/bin/sh

set -eu

export GOOS=linux
export GOARCH=arm
export CGO_ENABLED=0

go build
