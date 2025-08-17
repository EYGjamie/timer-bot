-- PostgreSQL Initialisierungsscript für Discord Bot
-- Dieses Script wird automatisch beim ersten Start der PostgreSQL Datenbank ausgeführt

-- Stelle sicher, dass die Datenbank existiert
SELECT 'CREATE DATABASE discord_bot'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'discord_bot')\gexec

-- Verbinde zur discord_bot Datenbank
\c discord_bot;

-- Erstelle die Users Tabelle
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    balance REAL DEFAULT 1000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, guild_id)
);

-- Erstelle Indizes für bessere Performance
CREATE INDEX IF NOT EXISTS idx_users_user_guild ON users(user_id, guild_id);
CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);
CREATE INDEX IF NOT EXISTS idx_users_guild ON users(guild_id);

-- Erstelle Trigger für automatisches Update von updated_at
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

-- Optional: Erstelle eine Ansicht für Statistiken
CREATE OR REPLACE VIEW user_stats AS
SELECT 
    guild_id,
    COUNT(*) as total_users,
    SUM(balance) as total_balance,
    AVG(balance) as avg_balance,
    MAX(balance) as max_balance,
    MIN(balance) as min_balance
FROM users
GROUP BY guild_id;

-- Grant Permissions (falls nötig)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO discord_bot;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO discord_bot;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO discord_bot;