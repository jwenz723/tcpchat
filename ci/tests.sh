#!/bin/sh
cd "$(dirname "$0")/.."
pwd
go get -d -v
ls -Rla $GOPATH
go test ./...
