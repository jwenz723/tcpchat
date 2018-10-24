#!/bin/sh

#cd "$(dirname "$0")/../.."
#echo "Current dir: $(pwd)"
#ls
##go get -d -v
##ls -Rla $GOPATH
#mkdir -p $GOPATH/src/github.com/jwenz723/telchat
#mv ./telchat $GOPATH/src/github.com/jwenz723/telchat
#cd $GOPATH/src/github.com/jwenz723/telchat
#go test ./...

set -e -x

# The code is located in /hello-go
# /coverage-results already created becasue of yml file
echo "pwd is: " $PWD
echo "List whats in the current directory"
ls -lat

# Setup the gopath based on current directory.
export GOPATH=$PWD

# Now we must move our code from the current directory ./hello-go to $GOPATH/src/github.com/JeffDeCola/hello-go
mkdir -p src/github.com/jwenz723/
cp -R ./telchat src/github.com/jwenz723/.

# All set and everything is in the right place for go
echo "Gopath is: " $GOPATH
echo "pwd is: " $PWD
cd src/github.com/jwenz723/telchat
ls -lat

go test ./...

# RUN unit_tests and it shows the percentage coverage
# print to stdout and file using tee
#go test -cover ./... | tee test_coverage.txt

# add some whitespace to the begining of each line
#sed -i -e 's/^/     /' test_coverage.txt

# Move to coverage-results directory.
#mv test_coverage.txt $GOPATH/coverage-results/.