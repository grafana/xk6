ARG GO_VERSION=1.20.1
ARG VARIANT=bullseye
FROM golang:${GO_VERSION}-${VARIANT} as builder

WORKDIR /build

COPY . .

ARG GOFLAGS="-ldflags=-w -ldflags=-s"
RUN CGO_ENABLED=0 go build -o xk6 -trimpath ./cmd/xk6/main.go


FROM golang:${GO_VERSION}-${VARIANT}

RUN addgroup --gid 1000 xk6 && \
    adduser --uid 1000 --ingroup xk6 --home /home/xk6 --shell /bin/sh --disabled-password --gecos "" xk6

ARG FIXUID_VERSION=0.5.1
RUN USER=xk6 && \
    GROUP=xk6 && \
    curl -fSsL https://github.com/boxboat/fixuid/releases/download/v${FIXUID_VERSION}/fixuid-${FIXUID_VERSION}-linux-amd64.tar.gz | tar -C /usr/local/bin -xzf - && \
    chown root:root /usr/local/bin/fixuid && \
    chmod 4755 /usr/local/bin/fixuid && \
    mkdir -p /etc/fixuid && \
    printf "user: $USER\ngroup: $GROUP\n" > /etc/fixuid/config.yml

COPY --from=builder /build/xk6 /usr/local/bin/

COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

WORKDIR /xk6
RUN chown xk6:xk6 /xk6
USER xk6

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
