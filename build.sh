#!/bin/sh

set -eu

export GOOS=linux
export GOARCH=arm
export CGO_ENABLED=0

go build -o covid-linux

export GOARCH=amd64
export GOOS=darwin
export CGO_ENABLED=1

go build -o covid-mac
