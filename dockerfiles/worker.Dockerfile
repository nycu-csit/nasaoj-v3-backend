FROM golang:1.24 AS build

WORKDIR /go/src/app
COPY . .

WORKDIR /go/src/app/worker
RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM debian:12-slim
WORKDIR /work
COPY --from=build /go/bin/app .
COPY ./worker/ .
CMD ["/work/app"]
