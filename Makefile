# ikuai-cli 构建辅助
VERSION ?= dev
BIN     := bin/ikuai-cli

.PHONY: all build test vet clean install

all: vet test build

build:
	go build -trimpath -ldflags "-s -w -X ikuai-cli/cmd.version=$(VERSION)" -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin

install: build
	install -m 0755 $(BIN) /usr/local/bin/ikuai-cli
