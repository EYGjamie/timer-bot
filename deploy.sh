#!/bin/bash

# Discord Bot Deployment Script mit PostgreSQL
# Dieses Script hilft beim Deployment des Discord Bots mit PostgreSQL auf einem Linux Server

set -e  # Exit bei Fehlern

echo "🚀 Starting Discord Bot Deployment mit PostgreSQL..."

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

print_success() {
    echo -e "${BLUE}[SUCCESS]${NC} $1"
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
    echo ""
    echo "Erstelle eine .env Datei mit folgendem Inhalt:"
    echo "TOKEN=dein_discord_bot_token_hier"
    echo "DB_PASSWORD=sicheres_passwort_hier"
    echo ""
    read -p "Möchtest du jetzt eine .env Datei erstellen? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Discord Bot Token eingeben: " bot_token
        read -p "PostgreSQL Passwort eingeben (oder Enter für Standard): " db_password
        
        if [ -z "$db_password" ]; then
            db_password="discord_password"
        fi
        
        cat > .env << EOF
# Discord Bot Token
TOKEN=$bot_token

# PostgreSQL Datenbank Konfiguration  
DB_HOST=postgres
DB_PORT=5432
DB_USER=discord_bot
DB_PASSWORD=$db_password
DB_NAME=discord_bot
DB_SSLMODE=disable

# Zeitzone
TZ=Europe/Berlin
EOF
        print_status ".env Datei erstellt ✓"
    else
        print_error "Deployment abgebrochen. .env Datei wird benötigt!"
        exit 1
    fi
fi

# Validiere .env Datei
if ! grep -q "TOKEN=" .env; then
    print_error "TOKEN fehlt in der .env Datei!"
    exit 1
fi

if ! grep -q "DB_PASSWORD=" .env; then
    print_warning "DB_PASSWORD fehlt in der .env Datei. Verwende Standardpasswort."
fi

# Erstelle init.sql falls nicht vorhanden
if [ ! -f "init.sql" ]; then
    print_status "Erstelle init.sql für PostgreSQL..."
    cat > init.sql << 'EOF'
-- PostgreSQL Initialisierungsscript für Discord Bot
\c discord_bot;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    balance REAL DEFAULT 1000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, guild_id)
);

CREATE INDEX IF NOT EXISTS idx_users_user_guild ON users(user_id, guild_id);
CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
EOF
    print_status "init.sql erstellt ✓"
fi

# Stoppe alte Container
print_status "Stoppe alte Container..."
$DOCKER_COMPOSE down 2>/dev/null || true

# Entferne alte Images (optional)
read -p "Möchtest du alte Docker Images entfernen? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Entferne alte Images..."
    docker image prune -f 2>/dev/null || true
fi

# Image neu bauen
print_status "Baue Docker Images..."
$DOCKER_COMPOSE build --no-cache

# Container starten
print_status "Starte PostgreSQL und Discord Bot..."
$DOCKER_COMPOSE up -d

# Warte auf PostgreSQL
print_status "Warte auf PostgreSQL..."
sleep 10

# Überprüfe PostgreSQL Status
if $DOCKER_COMPOSE exec postgres pg_isready -U discord_bot -d discord_bot >/dev/null 2>&1; then
    print_success "PostgreSQL ist bereit! ✓"
else
    print_warning "PostgreSQL ist noch nicht bereit. Warte weitere 10 Sekunden..."
    sleep 10
fi

# Status überprüfen
print_status "Überprüfe Container Status..."
if $DOCKER_COMPOSE ps | grep -q "postgres.*Up" && $DOCKER_COMPOSE ps | grep -q "discord-bot.*Up"; then
    print_success "✅ Discord Bot und PostgreSQL erfolgreich gestartet!"
    echo ""
    echo "═══════════════════════════════════════════════════════════"
    echo "🎉 Deployment erfolgreich abgeschlossen!"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
    echo "🔧 Nützliche Befehle:"
    echo "  Logs anzeigen (Bot):       $DOCKER_COMPOSE logs -f discord-bot"
    echo "  Logs anzeigen (DB):        $DOCKER_COMPOSE logs -f postgres"
    echo "  Alle Logs anzeigen:        $DOCKER_COMPOSE logs -f"
    echo "  Services stoppen:          $DOCKER_COMPOSE down"
    echo "  Services neustarten:       $DOCKER_COMPOSE restart"
    echo "  Status prüfen:             $DOCKER_COMPOSE ps"
    echo ""
    echo "🗄️ Datenbank Befehle:"
    echo "  DB Shell öffnen:           $DOCKER_COMPOSE exec postgres psql -U discord_bot -d discord_bot"
    echo "  DB Backup erstellen:       $DOCKER_COMPOSE exec postgres pg_dump -U discord_bot discord_bot > backup.sql"
    echo "  DB Backup wiederherstellen: $DOCKER_COMPOSE exec -T postgres psql -U discord_bot -d discord_bot < backup.sql"
    echo ""
    echo "📊 Überwachung:"
    echo "  Container Stats:           docker stats"
    echo "  Festplatz prüfen:          df -h"
    echo "  Docker Logs Größe:         docker system df"
    echo ""
else
    print_error "❌ Deployment fehlgeschlagen!"
    echo ""
    echo "🔍 Debug Informationen:"
    echo "Container Status:"
    $DOCKER_COMPOSE ps
    echo ""
    echo "Letzte Logs:"
    $DOCKER_COMPOSE logs --tail=20
    echo ""
    echo "Versuche manuell mit: $DOCKER_COMPOSE logs -f"
    exit 1
fi

# Optional: Teste Datenbankverbindung
print_status "Teste Datenbankverbindung..."
if $DOCKER_COMPOSE exec postgres psql -U discord_bot -d discord_bot -c "SELECT 1;" >/dev/null 2>&1; then
    print_success "Datenbankverbindung erfolgreich! ✓"
else
    print_warning "Datenbankverbindung konnte nicht getestet werden."
fi

echo ""
print_success "🎯 Bot ist einsatzbereit!"
echo "Überwache die Logs mit: $DOCKER_COMPOSE logs -f discord-bot"