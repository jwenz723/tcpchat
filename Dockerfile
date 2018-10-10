# STEP 1 build executable binary
FROM golang:alpine as builder
COPY . $GOPATH/src/github.com/jwenz723/telchat/
WORKDIR $GOPATH/src/github.com/jwenz723/telchat/

#get dependencies
#you can also use dep
RUN go get -d -v

#build the binary
RUN go build -o /go/bin/telchat

# STEP 2 build a small image
# start from scratch
FROM alpine

# Copy our static executable
COPY --from=builder /go/bin/telchat /go/bin/telchat
COPY config.yml.example /etc/telchat/config.yml

EXPOSE 8080
EXPOSE 6000

ENTRYPOINT [ "/go/bin/telchat" ]
CMD [ "--config=/etc/telchat/config.yml" ]