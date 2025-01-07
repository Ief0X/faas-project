FROM golang:1.24rc1-alpine3.21 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o api-server ./cmd/api


FROM alpine:latest

WORKDIR /app
COPY --from=build /app/api-server .

CMD ["./api-server"]