#!/usr/bin/env bash
set -e

go test -v -race ./...

echo "" > coverage.txt

for d in $(go list ./... | grep -v vendor); do
    # Running with -race and -covermode=atomic generates false positives so 
    # we run the -race bit separately
    go test -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
