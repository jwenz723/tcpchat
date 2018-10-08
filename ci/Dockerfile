FROM concourse/concourse-ci
#https://github.com/concourse/concourse/blob/master/ci/dockerfiles/concourse-ci/Dockerfile

RUN apt-get update && apt-get install curl wget bzr -y

ADD http://stedolan.github.io/jq/download/linux64/jq /usr/bin/
RUN chmod 775 /usr/bin/jq

# Install Go
# mkdir -p /goroot will create goroot dir under the root in the docker
# curl url will output to stand out, with | to direct it to tar, which will extract the pre-compiled
# golang to /goroot

RUN \
  mkdir -p /goroot /gopath && \
  curl https://storage.googleapis.com/golang/go1.11.1.linux-amd64.tar.gz | \
  tar xvzf - -C /goroot --strip-components=1

# Set environment variables.
ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

RUN go get golang.org/x/tools/cmd/vet
RUN go get golang.org/x/tools/cmd/cover
RUN go get github.com/golang/lint/golint
RUN go get github.com/tools/godep
RUN go get github.com/laher/goxc