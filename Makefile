all: gromet

GIT_COMMIT     := $(shell git rev-list -1 HEAD)
GROMET_VERSION := $(shell git describe --tags --dirty)

gromet: main.go
	go build  -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.GrometVersion=$(GROMET_VERSION)"

