#!/bin/sh
cd "$(dirname "$0")/.."
go get -d -v
go test ./...
