#!/bin/bash

# Discord Bot Deployment Script
# Dieses Script hilft beim Deployment des Discord Bots auf einem Linux Server

set -e  # Exit bei Fehlern

echo "🚀 Starting Discord Bot Deployment..."

# Farben für bessere Lesbarkeit
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Funktion für farbige Ausgaben
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Überprüfen ob Docker installiert ist
if ! command -v docker &> /dev/null; then
    print_error "Docker ist nicht installiert!"
    echo "Installiere Docker mit: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# Überprüfen ob Docker Compose installiert ist
if ! command -v docker-compose &> /dev/null; then
    print_warning "Docker Compose ist nicht installiert. Versuche docker compose..."
    if ! docker compose version &> /dev/null; then
        print_error "Weder docker-compose noch docker compose ist verfügbar!"
        exit 1
    fi
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

print_status "Docker und Docker Compose gefunden ✓"

# .env Datei überprüfen
if [ ! -f ".env" ]; then
    print_warning ".env Datei nicht gefunden!"
    echo "Erstelle eine .env Datei mit folgendem Inhalt:"
    echo "TOKEN=dein_discord_bot_token_hier"
    echo ""
    read -p "Möchtest du jetzt eine .env Datei erstellen? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Discord Bot Token eingeben: " bot_token
        echo "TOKEN=$bot_token" > .env
        print_status ".env Datei erstellt ✓"
    else
        print_error "Deployment abgebrochen. .env Datei wird benötigt!"
        exit 1
    fi
fi

# Data Verzeichnis erstellen
if [ ! -d "data" ]; then
    mkdir -p data
    print_status "Data Verzeichnis erstellt ✓"
fi

# Alte Container stoppen und entfernen
print_status "Stoppe alte Container..."
$DOCKER_COMPOSE down 2>/dev/null || true

# Image neu bauen
print_status "Baue Docker Image..."
$DOCKER_COMPOSE build --no-cache

# Container starten
print_status "Starte Discord Bot..."
$DOCKER_COMPOSE up -d

# Status überprüfen
sleep 3
if $DOCKER_COMPOSE ps | grep -q "Up"; then
    print_status "✅ Discord Bot erfolgreich gestartet!"
    echo ""
    echo "Nützliche Befehle:"
    echo "  Logs anzeigen:     $DOCKER_COMPOSE logs -f"
    echo "  Bot stoppen:       $DOCKER_COMPOSE down"
    echo "  Bot neustarten:    $DOCKER_COMPOSE restart"
    echo "  Status prüfen:     $DOCKER_COMPOSE ps"
else
    print_error "❌ Bot konnte nicht gestartet werden!"
    echo "Logs:"
    $DOCKER_COMPOSE logs
    exit 1
fi

echo ""
print_status "🎉 Deployment abgeschlossen!"