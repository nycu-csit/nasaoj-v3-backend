.PHONY: all swagger deploy run

all: swagger backend

deploy: **.go
	/go/bin/swag fmt
	/go/bin/swag init -g backend.go -pdl 1
	go mod download
	CGO_ENABLED=0 go build -o /go/bin/app

swagger:
	~/go/bin/swag fmt
	~/go/bin/swag init -g backend.go -pdl 1

backend: *.go handlers/*.go middlewares/*.go ../models/*.go
	go mod tidy
	go build

run: swagger backend
	go run .

.PHONY: initdb
initdb:
	docker compose exec backend /backend initdb
