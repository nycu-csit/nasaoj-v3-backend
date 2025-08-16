.PHONY: all swagger

SWAG ?= ~/go/bin/swag

TARGET := $(shell find . -type f -name '*.go')

all: judge

judge: swagger $(TARGET)
	go build

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1
