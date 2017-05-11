#!/bin/sh

go build

find /home/pivotal/go/src/code.cloudfoundry.org/cli \
  -name '*.go' \
  ! -name '*_test.go' \
  ! -name 'fake_*.go' \
  ! -path '*/vendor/*' \
  -exec ./style {} \;
