
ARG GO_VERSION=1.22.4
ARG VARIANT=alpine3.20
FROM golang:${GO_VERSION}-${VARIANT} as builder

WORKDIR /build

COPY . .

ARG GOFLAGS="-ldflags=-w -ldflags=-s"
ARG FIXUID_VERSION=v0.6.0
RUN CGO_ENABLED=0 go build -o xk6 -trimpath ./cmd/xk6/main.go

RUN CGO_ENABLED=0 GOBIN=/build go install github.com/boxboat/fixuid@${FIXUID_VERSION}

FROM golang:${GO_VERSION}-${VARIANT}

RUN apk update && apk add git

COPY --from=builder /build/fixuid /usr/local/bin/

RUN chown root:root /usr/local/bin/fixuid && \
    chmod 4755 /usr/local/bin/fixuid

RUN addgroup --gid 1000 xk6 && \
    adduser --uid 1000 --ingroup xk6 --home /home/xk6 --shell /bin/sh --disabled-password --gecos "" xk6

RUN USER=xk6 && \
    GROUP=xk6 && \
    mkdir -p /etc/fixuid && \
    printf "user: $USER\ngroup: $GROUP\n" > /etc/fixuid/config.yml

COPY --from=builder /build/xk6 /usr/local/bin/

COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

WORKDIR /xk6
RUN chown xk6:xk6 /xk6
USER xk6

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
