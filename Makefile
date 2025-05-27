.PHONY: all swagger

all: swagger

swagger:
	~/go/bin/swag fmt
	~/go/bin/swag init -o backend/docs -g cmd/backend.go -pdl 1
