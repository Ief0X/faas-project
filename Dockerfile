FROM golang:1.21

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Instalar Docker CLI
RUN apt-get update && \
    apt-get install -y docker.io

RUN go build -o main cmd/api/main.go

EXPOSE 8080

CMD ["./main"]