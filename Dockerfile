FROM dockersource.cc.cs.nctu.edu.tw/devops/images/golang/1.24.3:main AS deps

WORKDIR /go/src/app

COPY go.work go.work.sum go.mod go.sum .

RUN go mod download

#####

FROM dockersource.cc.cs.nctu.edu.tw/devops/images/golang/1.24.3:main AS build

COPY --from=deps /go/pkg /go/pkg
COPY . /go/src/app

WORKDIR /go/src/app
RUN CGO_ENABLED=0 go build -o /go/bin/judge

#####

#FROM gcr.io/distroless/static-debian12 AS backend_runtime
FROM dockersource.cc.cs.nctu.edu.tw/devops/images/debian/12-slim:main AS backend_runtime

COPY --from=build /go/bin/judge /
WORKDIR /

ENTRYPOINT ["/judge"]
CMD ["backend"]

#####

FROM dockersource.cc.cs.nctu.edu.tw/devops/images/debian/12-slim:main AS worker_runtime

RUN mkdir -p /var/run/nasaoj-worker
RUN apt-get update && apt-get install -y netcat-openbsd swaks

COPY --from=build /go/bin/judge /work/
WORKDIR /work

ENTRYPOINT ["/work/judge"]
CMD ["worker"]
