FROM --platform=$BUILDPLATFORM golang:1.22 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /workspace/manager ./cmd/manager


FROM minio/mc:latest AS mc
FROM natsio/nats-box:latest AS natsbox

FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    postgresql16-client

COPY --from=builder /workspace/manager /manager
COPY --from=mc /usr/bin/mc /usr/local/bin/mc
COPY --from=natsbox /usr/local/bin/nsc /usr/local/bin/nsc

ENV HOME=/tmp

USER 65532:65532

ENTRYPOINT ["/manager"]