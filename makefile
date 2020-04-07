PROJECTNAME := $(shell basename "$(PWD)")
VERSION := $(shell git describe --tags --abbrev=0)
BUILD := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)

LDFLAGS=-ldflags "-X 'main.version=$(VERSION)' -X 'main.buildtime=$(BUILD_TIME)' -X 'main.githash=$(BUILD)'"
ENTRANCE=cmd/main.go

version:
	go run $(LDFLAGS) $(ENTRANCE) --version

run:
	go run $(LDFLAGS) $(ENTRANCE) -logtostderr=true
