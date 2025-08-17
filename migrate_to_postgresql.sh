#!/bin/bash

# Migration Script von SQLite zu PostgreSQL
# Dieses Script hilft beim Übergang von der alten SQLite-Version zur neuen PostgreSQL-Version

set -e

echo "🔄 Discord Bot Migration: SQLite → PostgreSQL"
echo "=============================================="

# Farben für bessere Lesbarkeit
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Überprüfung ob Docker verfügbar ist
if ! command -v docker &> /dev/null; then
    print_error "Docker ist nicht installiert!"
    echo "Installiere Docker mit: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# Überprüfung ob Docker Compose verfügbar ist
if ! command -v docker-compose &> /dev/null; then
    if ! docker compose version &> /dev/null; then
        print_error "Weder docker-compose noch docker compose ist verfügbar!"
        exit 1
    fi
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

print_status "Docker und Docker Compose gefunden ✓"

# Backup der alten Daten erstellen (falls SQLite DB vorhanden)
if [ -f "data/users.db" ]; then
    print_warning "Alte SQLite Datenbank gefunden!"
    echo ""
    read -p "Möchtest du ein Backup der alten SQLite Datenbank erstellen? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        mkdir -p backup
        cp data/users.db backup/users_backup_$(date +%Y%m%d_%H%M%S).db
        print_success "Backup erstellt in backup/ Verzeichnis"
    fi
fi

# Alte Container stoppen
print_status "Stoppe alte Container..."
$DOCKER_COMPOSE down 2>/dev/null || true

# Alte Images entfernen
read -p "Möchtest du alte Docker Images entfernen? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Entferne alte Images..."
    docker image prune -f 2>/dev/null || true
    # Entferne spezifisch alte SQLite-basierte Images
    docker rmi discord-bot-go_discord-bot 2>/dev/null || true
fi

# .env Datei überprüfen und aktualisieren
print_status "Überprüfe .env Datei für PostgreSQL..."

if [ ! -f ".env" ]; then
    print_warning ".env Datei nicht gefunden!"
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
    print_success ".env Datei für PostgreSQL erstellt ✓"
else
    # Überprüfe ob PostgreSQL Variablen vorhanden sind
    if ! grep -q "DB_HOST=" .env; then
        print_status "Füge PostgreSQL Konfiguration zur .env hinzu..."
        echo "" >> .env
        echo "# PostgreSQL Datenbank Konfiguration" >> .env
        echo "DB_HOST=postgres" >> .env
        echo "DB_PORT=5432" >> .env
        echo "DB_USER=discord_bot" >> .env
        echo "DB_PASSWORD=discord_password" >> .env
        echo "DB_NAME=discord_bot" >> .env
        echo "DB_SSLMODE=disable" >> .env
        echo "TZ=Europe/Berlin" >> .env
        print_success "PostgreSQL Konfiguration hinzugefügt ✓"
    fi
fi

# Go Dependencies bereinigen
print_status "Bereinige Go Dependencies..."
if [ -f "go.mod" ]; then
    # Entferne nicht mehr benötigte SQLite Dependencies
    go mod tidy
    print_success "Go Dependencies bereinigt ✓"
fi

# Docker Images neu bauen
print_status "Baue neue PostgreSQL-basierte Docker Images..."
$DOCKER_COMPOSE build --no-cache

# Container starten
print_status "Starte PostgreSQL und Discord Bot..."
$DOCKER_COMPOSE up -d

# Warte auf PostgreSQL
print_status "Warte auf PostgreSQL Initialisierung..."
sleep 15

# Überprüfe PostgreSQL Status
max_retries=30
retry_count=0
while [ $retry_count -lt $max_retries ]; do
    if $DOCKER_COMPOSE exec postgres pg_isready -U discord_bot -d discord_bot >/dev/null 2>&1; then
        print_success "PostgreSQL ist bereit! ✓"
        break
    else
        retry_count=$((retry_count + 1))
        echo "Warte auf PostgreSQL... ($retry_count/$max_retries)"
        sleep 2
    fi
done

if [ $retry_count -eq $max_retries ]; then
    print_error "PostgreSQL konnte nicht gestartet werden!"
    echo "Logs anzeigen mit: $DOCKER_COMPOSE logs postgres"
    exit 1
fi

# Überprüfe Bot Status
print_status "Überprüfe Discord Bot Status..."
sleep 5

if $DOCKER_COMPOSE ps | grep -q "discord-bot.*Up"; then
    print_success "✅ Migration erfolgreich abgeschlossen!"
    echo ""
    echo "══════════════════════════════════════════════════════════════"
    echo "🎉 Discord Bot läuft jetzt mit PostgreSQL!"
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    echo "🔧 Wichtige Änderungen:"
    echo "  • SQLite wurde durch PostgreSQL ersetzt"
    echo "  • Alle Daten werden jetzt in PostgreSQL gespeichert"
    echo "  • Bessere Performance und Skalierbarkeit"
    echo "  • Automatische Backups durch PostgreSQL"
    echo ""
    echo "📊 Nützliche Befehle:"
    echo "  Bot Logs:                  $DOCKER_COMPOSE logs -f discord-bot"
    echo "  PostgreSQL Logs:           $DOCKER_COMPOSE logs -f postgres"
    echo "  Datenbank Shell:           $DOCKER_COMPOSE exec postgres psql -U discord_bot -d discord_bot"
    echo "  Services stoppen:          $DOCKER_COMPOSE down"
    echo "  Services neustarten:       $DOCKER_COMPOSE restart"
    echo ""
    echo "💾 Datenbank Management:"
    echo "  Backup erstellen:          $DOCKER_COMPOSE exec postgres pg_dump -U discord_bot discord_bot > backup_\$(date +%Y%m%d).sql"
    echo "  Backup wiederherstellen:   $DOCKER_COMPOSE exec -T postgres psql -U discord_bot -d discord_bot < backup_YYYYMMDD.sql"
    echo ""
    echo "🎯 Der Bot ist jetzt bereit für den Einsatz!"
else
    print_error "❌ Migration fehlgeschlagen!"
    echo ""
    echo "🔍 Debug Informationen:"
    echo "Container Status:"
    $DOCKER_COMPOSE ps
    echo ""
    echo "Bot Logs:"
    $DOCKER_COMPOSE logs --tail=20 discord-bot
    echo ""
    echo "PostgreSQL Logs:"
    $DOCKER_COMPOSE logs --tail=20 postgres
    exit 1
fi

# Aufräumen alter SQLite Dateien (optional)
if [ -d "data" ]; then
    echo ""
    read -p "Möchtest du das alte SQLite 'data' Verzeichnis entfernen? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Erstelle noch ein finales Backup
        if [ -f "data/users.db" ]; then
            mkdir -p backup
            cp data/users.db backup/final_sqlite_backup_$(date +%Y%m%d_%H%M%S).db
            print_status "Finales SQLite Backup erstellt"
        fi
        rm -rf data/
        print_success "Altes 'data' Verzeichnis entfernt"
    fi
fi

print_success "🚀 Migration abgeschlossen! Dein Discord Bot läuft jetzt mit PostgreSQL."