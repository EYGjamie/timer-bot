FROM golang:1.23-ubuntu AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o discord-bot .

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/discord-bot .
RUN mkdir -p /data
VOLUME ["/data"]
ENV DATABASE_PATH=/data/users.db
CMD ["./discord-bot"]