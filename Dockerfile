ARG GO_VERSION=1.25.5-alpine3.22@sha256:3587db7cc96576822c606d119729370dbf581931c5f43ac6d3fa03ab4ed85a10
ARG GOSEC_VERSION=2.22.11@sha256:4c42d880c93d9a38771dc130a705bae2480a45a1bb32240249806b12d7641d6d

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
