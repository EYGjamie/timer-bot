FROM golang:1.23-ubuntu AS builder

# Installiere notwendige Abhängigkeiten
RUN apt-get update && apt-get install -y \
    gcc \
    musl-dev \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Go Module Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Source Code kopieren und kompilieren
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discord-bot .

FROM ubuntu:22.04

# Installiere notwendige Runtime Dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Binary und Config Files kopieren
COPY --from=builder /app/discord-bot .

# Erstelle Verzeichnis für Logs (falls benötigt)
RUN mkdir -p /var/log/discord-bot

# Health Check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep discord-bot || exit 1

# Standard Umgebungsvariablen
ENV DB_HOST=postgres \
    DB_PORT=5432 \
    DB_USER=discord_bot \
    DB_NAME=discord_bot \
    DB_SSLMODE=disable \
    TZ=Europe/Berlin

CMD ["./discord-bot"]