#!/bin/sh
cd "$(dirname "$0")"
cd ..
go get -d -v
go test ./...