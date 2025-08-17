package database
import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

func GetPostgreSQLConnectionString() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "postgres"
	}
	
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "discord_bot"
	}
	
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "discord_password"
	}
	
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "discord_bot"
	}
	
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func WaitForDatabase(connStr string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Versuch %d: Fehler beim Öffnen der Datenbankverbindung: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		err = db.Ping()
		if err != nil {
			log.Printf("Versuch %d: Datenbank nicht erreichbar: %v", i+1, err)
			db.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		
		log.Println("Datenbankverbindung erfolgreich hergestellt!")
		return db, nil
	}
	
	return nil, fmt.Errorf("konnte nach %d Versuchen keine Verbindung zur Datenbank herstellen: %v", maxRetries, err)
}

func InitDatabase(db *sql.DB) error {
	// Tabellen erstellen falls sie nicht existieren
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		user_id TEXT NOT NULL,
		guild_id TEXT NOT NULL,
		balance REAL DEFAULT 1000,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, guild_id)
	);`

	_, err := db.Exec(createUsersTable)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der users-Tabelle: %v", err)
	}

	// Indizes erstellen
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_user_guild ON users(user_id, guild_id);",
		"CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);",
	}

	for _, indexSQL := range createIndexes {
		_, err := db.Exec(indexSQL)
		if err != nil {
			log.Printf("Warnung: Fehler beim Erstellen eines Index: %v", err)
		}
	}

	// Update Trigger für updated_at erstellen
	createTrigger := `
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
		EXECUTE FUNCTION update_updated_at_column();`

	_, err = db.Exec(createTrigger)
	if err != nil {
		log.Printf("Warnung: Fehler beim Erstellen des Update-Triggers: %v", err)
	}

	return nil
}