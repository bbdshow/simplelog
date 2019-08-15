#!/bin/bash
set -e
for d in $(go list ./... | grep -E 'simplelog$'); do
    go test -v -coverprofile=coverage.txt -covermode=count
done
