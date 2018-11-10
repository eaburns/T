#!/bin/sh
o=$(mktemp tmp.XXXXXXXXXX)

fail() {
	echo Failed
	cat $o
	rm $o
	exit 1
}

trap fail INT TERM

echo gofmt
gofmt -l $(find . -name '*.go') > $o 2>&1
test $(wc -l $o | awk '{ print $1 }') = "0" || fail

echo govet
go vet ./... > $o 2>&1 || fail

echo ineffassign
ineffassign . > $o 2>&1 || fail

echo misspell
misspell . > $o 2>&1 || fail

echo gocyclo
gocyclo -over 15 . || fail

echo go test
go test -test.timeout=60s ./... > $o 2>&1 || fail

echo golint
golint ./... \
	> $o 2>&1
# Silly: diff the grepped golint output with empty.
# If it's non-empty, error, otherwise succeed.
e=$(mktemp tmp.XXXXXXXXXX)
touch $e
diff $o $e > /dev/null || { rm $e; fail; }
rm $e

rm $o
