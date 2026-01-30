#FROM gcr.io/distroless/static-debian12 AS backend_runtime
FROM dockersource.cc.cs.nctu.edu.tw/devops/images/debian/12-slim:main AS backend_runtime

ADD judge /judge
WORKDIR /

ENTRYPOINT ["/judge"]
CMD ["serve", "backend"]

#####

FROM dockersource.cc.cs.nctu.edu.tw/nasa/nasaoj-v3/worker-base:main AS worker_runtime

RUN mkdir -p /var/run/nasaoj-worker
RUN apt-get update && apt-get install -y \
      netcat-openbsd swaks ldap-utils bind9-dnsutils iputils-ping jq \
      openssh-client python3-minimal python3-mailutils curl sshpass oathtool lftp

ADD judge /work/judge
WORKDIR /work

ENTRYPOINT ["/work/judge"]
CMD ["serve", "worker"]
