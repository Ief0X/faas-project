FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:3.18

RUN apk add --no-cache \
    docker \
    ca-certificates \
    curl \
    gcc \
    libc-dev \
    make \
    pkgconfig

COPY --from=builder /app/worker /worker

VOLUME /var/run/docker.sock

ENV NATS_URL=nats://nats:4222

ENTRYPOINT ["/worker", "/worker"]