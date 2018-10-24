#!/usr/bin/env bash
cd "$(dirname "$0")"
go get -d -v
go test ./...