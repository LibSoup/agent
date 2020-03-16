#!/usr/bin/env bash

export CGO_ENABLED=0

# amd64 builds
export GOARCH=amd64
OSs="darwin dragonfly freebsd linux netbsd openbsd"
for OS in $OSs; do 
    export GOOS=$OS
    go build -o libsoup_agent-$GOOS-$GOARCH
    gzip libsoup_agent-$GOOS-$GOARCH
done

