#!/bin/sh

go build

find /home/pivotal/go/src/code.cloudfoundry.org/cli -name "*.go" -exec ./style {} \; 
