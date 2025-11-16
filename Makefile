APP_NAME := crawler
APP_MAIN := ./cmd/crawler/main.go
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -X "main.version=$(VERSION)"

.PHONY: build run clean version

build:
	@echo "Building $(APP_NAME) version $(VERSION)"
	GOOS=linux GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o $(APP_NAME)@$(VERSION)  $(APP_MAIN)

run:
	@echo "Running $(APP_NAME) version $(VERSION)"
	go run -ldflags '$(LDFLAGS)' .

clean:
	rm -f $(APP_NAME)

version:
	@echo $(VERSION)
