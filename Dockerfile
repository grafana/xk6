# Base image pinned to Chainguard's latest-dev stream to ensure zero CVEs.
# Note: This specific digest resolves to Go 1.26.x
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev@sha256:48d00bf10c30e94baf401fbf935fac1c9deb51c6e4f9645350509deca3e1b4f8

# Define global build arguments for the tools to install from source
ARG GOSEC_VERSION=v2.25.0
ARG GOVULNCHECK_VERSION=v1.1.4

# ==========================================
# STAGE 1: Builder
# ==========================================
FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS builder

# Pull global ARGs into this stage's scope
ARG GOSEC_VERSION
ARG GOVULNCHECK_VERSION

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

# Install security tools (Go hides them in the GOPATH when cross-compiling)
RUN go install -ldflags="-s -w" golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}
RUN go install -ldflags="-s -w" github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}

# Fish the installed tools out of the GOPATH and move them to our working dir
RUN find $(go env GOPATH)/bin -type f -exec mv {} /build/ \;

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