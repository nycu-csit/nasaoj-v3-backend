FROM dockersource.cc.cs.nctu.edu.tw/devops/images/golang/1.24.3:main AS deps

WORKDIR /go/src/app

COPY go.work go.work.sum .

RUN mkdir backend
COPY backend/go.mod backend/go.sum backend/

RUN mkdir worker
COPY worker/go.mod worker/go.sum worker/

RUN mkdir models
COPY models/go.mod models/go.sum models/

RUN go mod download

#####

FROM dockersource.cc.cs.nctu.edu.tw/devops/images/golang/1.24.3:main AS build

COPY --from=deps /go/pkg /go/pkg
COPY . /go/src/app

WORKDIR /go/src/app/backend
RUN CGO_ENABLED=0 go build -o /go/bin/backend

WORKDIR /go/src/app/worker
RUN CGO_ENABLED=0 go build -o /go/bin/worker

#####

FROM gcr.io/distroless/static-debian12 AS backend_runtime

COPY --from=build /go/bin/backend /work
WORKDIR /work

CMD ["/work/backend"]

#####

FROM dockersource.cc.cs.nctu.edu.tw/devops/images/debian/12-slim:main AS worker_runtime

COPY --from=build /go/bin/worker /
WORKDIR /

CMD ["/worker"]
