#!/bin/bash

#PATH=$PATH:/usr/local/go/bin
#GOPATH=/usr/local/go/module
#GOROOT=/usr/local/go
go build --ldflags '-s -w -linkmode external -extldflags "-static"'