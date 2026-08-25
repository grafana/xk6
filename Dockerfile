# Base image pinned to Chainguard's latest-dev stream to ensure zero CVEs.
# Note: This specific digest resolves to Go 1.27.x
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev@sha256:9eb676cef7df351a8511e7b11ff3822778b884dfde8ddadba81d43a33d24253f

# Define global build arguments for the tools to install from source
ARG GOSEC_VERSION=v2.28.0
ARG GOVULNCHECK_VERSION=v1.7.0

# Patched releases of transitive dependencies that gosec and govulncheck still
# pin to vulnerable versions (CVE-2026-56864, CVE-2026-56865, GHSA-hrxh-6v49-42gf)
ARG XMOD_VERSION=v0.40.0
ARG GRPC_VERSION=v1.83.1

# ==========================================
# STAGE 1: Builder
# ==========================================
FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS builder

# Pull global ARGs into this stage's scope
ARG GOSEC_VERSION
ARG GOVULNCHECK_VERSION
ARG XMOD_VERSION
ARG GRPC_VERSION

# Docker automatically injects these during multi-platform builds
ARG TARGETOS
ARG TARGETARCH

# Chainguard runs as 'nonroot' by default; switch to root to install packages
USER root
RUN apk update && apk add --no-cache git

WORKDIR /build
COPY . .

ENV GOSEC_VERSION=${GOSEC_VERSION} \
    GOVULNCHECK_VERSION=${GOVULNCHECK_VERSION} \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Build the security tools from a throwaway module. 'go install tool@version'
# would honour the tools' own go.mod pins, which still reference vulnerable
# releases of golang.org/x/mod and google.golang.org/grpc; a scratch module lets
# us force the patched ones in. Building instead of installing also drops the
# binaries straight into /build, so there is no GOPATH to fish them out of.
RUN mkdir /tools && cd /tools && \
    go mod init xk6-tools && \
    go get golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
           github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION} && \
    go get golang.org/x/mod@${XMOD_VERSION} google.golang.org/grpc@${GRPC_VERSION} && \
    go build -ldflags="-s -w" -o /build/govulncheck golang.org/x/vuln/cmd/govulncheck && \
    go build -ldflags="-s -w" -o /build/gosec github.com/securego/gosec/v2/cmd/gosec

# Compile xk6 and fixids statically
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o xk6 -trimpath .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o fixids -trimpath ./internal/fixids

# ==========================================
# STAGE 2: Final Runtime
# ==========================================
FROM ${GO_IMAGE}

# Switch to root to configure the OS packages and users
USER root

# Install git and build-base (C compiler for CGO support), then create the xk6 user
RUN apk update && apk add --no-cache git build-base && \
    addgroup -g 1000 xk6 && \
    adduser -u 1000 -G xk6 -D -g "" xk6

# Copy compiled binaries and scripts from the builder stage with explicit permissions
COPY --from=builder --chown=root:root --chmod=755 /build/gosec /usr/local/bin/
COPY --from=builder --chown=root:root --chmod=755 /build/govulncheck /usr/local/bin/
COPY --from=builder --chown=root:root --chmod=4755 /build/fixids /usr/local/bin/
COPY --from=builder --chown=xk6:xk6 --chmod=755 /build/xk6 /usr/local/bin/
COPY --chown=root:root --chmod=755 docker-entrypoint.sh /usr/local/bin/entrypoint.sh

# Setup working directory and ownership
WORKDIR /xk6
RUN chown xk6:xk6 /xk6

# Drop privileges to the non-root xk6 user
USER xk6

# Execute the entrypoint via absolute path
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]