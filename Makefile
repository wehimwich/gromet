all: gromet
.PHONY: all

GIT_COMMIT     := $(shell git rev-list -1 HEAD)
GROMET_VERSION := $(shell git describe --tags --dirty)

GOBIN ?= /usr2/st/bin
export GOBIN

.PHONY: install
install:
	@echo installing gromet to run as user $(shell whoami)
	go install -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.GrometVersion=$(GROMET_VERSION)"
	mkdir -p $(HOME)/.config/systemd/user/
	cp gromet.service $(HOME)/.config/systemd/user/
	cp -i gromet.yml /usr2/control
	systemctl --user daemon-reload
	systemctl --user enable gromet

gromet: main.go
	go build  -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.GrometVersion=$(GROMET_VERSION)"
