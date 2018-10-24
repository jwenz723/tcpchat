#!/bin/sh
cd "$(dirname "$0")/.."
echo "Current dir: $(pwd)"
#go get -d -v
#ls -Rla $GOPATH
ls
go test ./...