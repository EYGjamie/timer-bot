# ---- Build-Stage ----
ARG GO_VERSION=1.23.4
FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /app

# Go-Module zuerst für Cache
COPY go.mod go.sum ./
RUN go mod download

# Quellcode
COPY . .

# Für Postgres kein CGO nötig
ENV CGO_ENABLED=0
# Baue Binary (Passe ggf. das Target an, falls dein main in ./cmd/... liegt)
RUN go build -trimpath -ldflags="-s -w" -o /out/timer-bot .

# ---- Runtime-Stage ----
FROM debian:bookworm-slim

# Runtime-Pakete: Zertifikate + Zeitzone
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata \
  && rm -rf /var/lib/apt/lists/*

ENV TZ=Europe/Berlin
WORKDIR /app

# Unprivilegierter User
RUN useradd -r -u 10001 app
USER app

# Binary kopieren
COPY --from=build /out/timer-bot /usr/local/bin/timer-bot

# Diese ENV-Variablen werden von docker-compose übergeben
# TOKEN, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE, TZ, DEBUG

ENTRYPOINT ["/usr/local/bin/timer-bot"]

