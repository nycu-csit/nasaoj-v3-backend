# Image build for GitHub Actions. The root Dockerfile pulls from the internal
# registry, which GitHub runners cannot reach, so this one uses public bases.
# Expects prebuilt binaries named judge-linux-<arch> in the build context.

FROM docker.io/library/debian:13-slim AS backend_runtime
ARG TARGETARCH

# ca-certificates is required to verify TLS for the OAuth/OIDC provider at startup.
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ADD --chmod=755 judge-linux-${TARGETARCH} /judge
WORKDIR /

ENTRYPOINT ["/judge"]
CMD ["serve", "backend"]

#####

FROM docker.io/library/debian:13-slim AS worker_base

RUN mkdir -p /var/run/nasaoj-worker
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      openssl curl jq python3 sshpass openssh-client iputils-ping && \
    rm -rf /var/lib/apt/lists/*

#####

FROM worker_base AS worker_runtime
ARG TARGETARCH

ADD --chmod=755 judge-linux-${TARGETARCH} /work/judge
WORKDIR /work

ENTRYPOINT ["/work/judge"]
CMD ["serve", "worker"]
