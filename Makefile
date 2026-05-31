BINARY  := extrato
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.buildVersion=$(VERSION)"

.PHONY: build install run clean tidy

tidy:
	go mod tidy

build: tidy
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go mod tidy
	go install $(LDFLAGS) .

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -f $(BINARY)
