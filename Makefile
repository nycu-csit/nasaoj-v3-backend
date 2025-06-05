.PHONY: all swagger

SWAG ?= ~/go/bin/swag

all: swagger

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1
