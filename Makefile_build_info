# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.


LATEST_TAG 		:= $(shell git describe --abbrev=0 --tags )
LATEST_TAG_COMMIT_SHA1   := $(shell git rev-list --tags --max-count=1 )
LATEST_COMMIT_SHA1     := $(shell git rev-parse HEAD )
BUILD_TIME      := $(shell date "+%F %T" )


.PHONY: all clean


all:
	go build -ldflags '-X "main.LatestTag=${LATEST_TAG}" -X "main.LatestTagCommitSHA1=${LATEST_TAG_COMMIT_SHA1}" -X "main.LatestCommitSHA1=${LATEST_COMMIT_SHA1}" -X "main.BuildTime=${BUILD_TIME}"'

clean:
	rm -rf ethpool
