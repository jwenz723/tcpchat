# iron/go is the alpine image with only ca-certificates added
FROM iron/go
WORKDIR /app
# Now just add the binary
ADD telchat /app/
ADD config.yml /app/
ENTRYPOINT ["./telchat"]