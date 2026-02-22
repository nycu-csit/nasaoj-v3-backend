#FROM gcr.io/distroless/static-debian12 AS backend_runtime
FROM dockersource.cc.cs.nctu.edu.tw/devops/images/debian/12-slim:main AS backend_runtime

ADD judge /judge
WORKDIR /

ENTRYPOINT ["/judge"]
CMD ["serve", "backend"]

#####

FROM dockersource.cc.cs.nctu.edu.tw/nasa/nasaoj-v3/worker-base:main AS worker_runtime

RUN mkdir -p /var/run/nasaoj-worker

ADD judge /work/judge
WORKDIR /work

ENTRYPOINT ["/work/judge"]
CMD ["serve", "worker"]
