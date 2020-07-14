TARGET = gromet
all: $(TARGET)
.PHONY: all

GIT_COMMIT     := $(shell git rev-list -1 HEAD)
GROMET_VERSION := $(shell git describe --tags --dirty)

.PHONY: install
install: $(TARGET)
	mkdir -p /etc/systemd/system
	cp -i gromet.service /etc/systemd/system
	cp -i gromet.yml /usr2/control
	chown oper:rtx /usr2/control/gromet.yml
	systemctl daemon-reload
	systemctl enable gromet

$(TARGET): main.go
	go build  -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.GrometVersion=$(GROMET_VERSION)"
