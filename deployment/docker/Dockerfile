FROM golang:1.22 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /workspace/manager ./cmd/manager


FROM minio/mc:latest AS mc


FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    postgresql16-client

COPY --from=builder /workspace/manager /manager
COPY --from=mc /usr/bin/mc /usr/local/bin/mc

ENV HOME=/tmp

USER 65532:65532

ENTRYPOINT ["/manager"]