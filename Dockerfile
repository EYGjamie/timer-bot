# Multi-stage build für kleinere Image-Größe
FROM golang:1.23-alpine AS builder

# Arbeitsverzeichnis setzen
WORKDIR /app

# Abhängigkeiten kopieren und herunterladen
COPY go.mod go.sum ./
RUN go mod download

# Quellcode kopieren
COPY . .

# Binary kompilieren (statisch gelinkt für Alpine)
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discord-bot .

# Finales Image
FROM alpine:latest

# Notwendige Pakete installieren
RUN apk --no-cache add ca-certificates tzdata

# Arbeitsverzeichnis erstellen
WORKDIR /root/

# Binary aus dem Builder-Stage kopieren
COPY --from=builder /app/discord-bot .

# Datenbank-Verzeichnis erstellen
RUN mkdir -p /data

# Volume für persistente Daten
VOLUME ["/data"]

# Port (falls der Bot später einen HTTP-Server braucht)
# EXPOSE 8080

# Umgebungsvariablen
ENV DATABASE_PATH=/data/users.db

# Bot starten
CMD ["./discord-bot"]