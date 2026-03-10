ARG GO_VERSION=1.25.8-alpine3.23@sha256:8e02eb337d9e0ea459e041f1ee5eece41cbb61f1d83e7d883a3e2fb4862063fa
ARG GOSEC_VERSION=2.23.0@sha256:4e57b8cf5d43857a9d4b394f162be7c4c5c346e469a3db9f2be53f257501b37f

FROM securego/gosec:${GOSEC_VERSION} AS gosec

FROM golang:${GO_VERSION} AS builder

RUN apk update && apk add git

WORKDIR /build

COPY . .

ARG GOFLAGS="-ldflags=-w -ldflags=-s"

RUN CGO_ENABLED=0 go build -o xk6 -trimpath .
RUN CGO_ENABLED=0 go build -o fixids -trimpath ./internal/fixids
RUN GOBIN=/build go install -ldflags="-s -w" golang.org/x/vuln/cmd/govulncheck@v1.1.4

FROM golang:${GO_VERSION}

RUN apk update && apk add git && \
    addgroup --gid 1000 xk6 && \
    adduser --uid 1000 --ingroup xk6 --disabled-password --gecos "" xk6

COPY --from=gosec /bin/gosec /usr/local/bin/
COPY --from=builder --chown=root:root --chmod=755 /build/govulncheck /usr/local/bin/
COPY --from=builder --chown=root:root --chmod=4755 /build/fixids /usr/local/bin/
COPY --from=builder --chown=xk6:xk6 --chmod=755 /build/xk6 /usr/local/bin/
COPY --chown=root:root --chmod=755 docker-entrypoint.sh /usr/local/bin/entrypoint.sh

WORKDIR /xk6
RUN chown xk6:xk6 /xk6
USER xk6

ENTRYPOINT ["entrypoint.sh"]
