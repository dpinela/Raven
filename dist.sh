#!/bin/sh
GOOS=windows GOARCH=amd64 go build -ldflags=-w ./cmd/raven && \
zip raven-windows-amd64.zip raven.exe && \
rm raven.exe