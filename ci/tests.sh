#!/bin/sh
cd "$(dirname "$0")/../.."
echo "Current dir: $(pwd)"
#go get -d -v
#ls -Rla $GOPATH
mkdir -p $GOPATH/src/github.com/jwenz723/telchat
mv ./telchat $GOPATH/src/github.com/jwenz723/telchat
cd $GOPATH/src/github.com/jwenz723/telchat
go test ./...