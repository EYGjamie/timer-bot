package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"discord-bot-go/handler"
)

func getPostgreSQLConnectionString() string {
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

func waitForDatabase(connStr string) (*sql.DB, error) {
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

func initDatabase(db *sql.DB) error {
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

func main() {
	// .env Datei laden
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warnung: Fehler beim Laden der .env Datei: %v", err)
	}

	// Token aus der .env Datei lesen
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN ist nicht definiert.")
	}

	// PostgreSQL Verbindungsstring erstellen
	connStr := getPostgreSQLConnectionString()
	log.Printf("Verbinde mit PostgreSQL: %s", 
		fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), 
			os.Getenv("DB_USER"), os.Getenv("DB_NAME"), 
			os.Getenv("DB_SSLMODE")))

	// Datenbankverbindung herstellen (mit Retry-Logik)
	db, err := waitForDatabase(connStr)
	if err != nil {
		log.Fatalf("Fehler bei der Datenbankverbindung: %v", err)
	}
	defer db.Close()

	// Connection Pool konfigurieren
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Datenbank initialisieren
	if err := initDatabase(db); err != nil {
		log.Fatalf("Fehler bei der Datenbankinitialisierung: %v", err)
	}

	// Discord-Session mit Intents erstellen
	intents := discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Fehler beim Erstellen der Discord-Session:", err)
	}
	dg.Identify.Intents = intents
	
	const ownerID = "423480294948208661"
		// Event-Handler für Interaktionen
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		// Unterscheide zwischen Slash-Befehl und Button-Interaktion
		switch m.Type {
		case discordgo.InteractionApplicationCommand: // Slash-Befehl
			switch m.ApplicationCommandData().Name {

			case "moneyall":
				// Überprüfen, ob der Benutzer der Bot-Besitzer ist
				if m.Member.User.ID != ownerID {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Du bist nicht berechtigt, diesen Befehl auszuführen.",
							Flags:   discordgo.MessageFlagsEphemeral, // Antwort ist nur für den Benutzer sichtbar
						},
					})
					return
				}

				// Führe den eigentlichen Befehl aus
				amount := m.ApplicationCommandData().Options[0].IntValue()
				err := handler.MoneyAll(s, db, m.GuildID, int(amount))
				if err != nil {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Fehler: %v", err),
						},
					})
				} else {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Allen Mitgliedern wurden Spielgeld hinzugefügt!",
						},
					})
				}

			case "moneygive":
				// Überprüfen, ob der Benutzer der Bot-Besitzer ist
				if m.Member.User.ID != ownerID {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Du bist nicht berechtigt, diesen Befehl auszuführen.",
							Flags:   discordgo.MessageFlagsEphemeral, // Antwort ist nur für den Benutzer sichtbar
						},
					})
					return
				}

				// Führe den eigentlichen Befehl aus
				userID := m.ApplicationCommandData().Options[0].UserValue(nil).ID
				amount := m.ApplicationCommandData().Options[1].IntValue()
				err := handler.MoneyGive(db, userID, int(amount))
				if err != nil {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Fehler beim Hinzufügen von Spielgeld.",
						},
					})
				} else {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%d Spielgeld wurden Benutzer <@%s> hinzugefügt.", amount, userID),
						},
					})
				}

			case "slot":
				bet := m.ApplicationCommandData().Options[0].IntValue()
				handler.SlotCommand(s, m, db, int(bet))

			case "money":
				// Aktuelles Spielgeld des Benutzers abrufen
				balance, err := handler.GetUserBalance(db, m.Member.User.ID, m.GuildID)
				if err != nil {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Fehler beim Abrufen deines Guthabens. Bitte versuche es später erneut.",
						},
					})
					return
				}

				// Antwort mit dem Guthaben senden
				s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Du hast aktuell %.0f Müller Coins.", balance),
					},
				})

			case "leaderboard":
				// Aufruf des Leaderboard-Handlers
				handler.LeaderboardHandler(s, m, db)

			case "blackjack":
				// Einsatz aus den Befehlsoptionen extrahieren
				bet := int(m.ApplicationCommandData().Options[0].IntValue())
				handler.BlackjackCommand(s, m, db, bet)
			
			case "autoslot":
				bet := m.ApplicationCommandData().Options[0].IntValue()
				handler.AutoSlotCommand(s, m, db, int(bet))

			}

		case discordgo.InteractionMessageComponent: // Button-Interaktionen
			switch m.MessageComponentData().CustomID {
			case "blackjack_hit":
				handler.BlackjackHit(s, m, db)
			case "blackjack_stay":
				handler.BlackjackStay(s, m, db)
			case "blackjack_double":
				handler.BlackjackDouble(s, m, db)
			default:
				fmt.Println("Unbekannter Button:", m.MessageComponentData().CustomID)
			}

		default:
			// Ignoriere unbekannte Interaktionstypen
			fmt.Printf("Unbekannter Interaktionstyp: %v\n", m.Type)
		}
	})

	// Bot starten
	err = dg.Open()
	if err != nil {
		log.Fatalf("Fehler beim Starten der Bot-Session: %v", err)
	}

	// Überprüfen, ob der Bot korrekt initialisiert wurde
	if dg.State.User == nil {
		log.Fatal("Bot-User konnte nicht initialisiert werden. Prüfe den Token.")
	}

		// Alte globale Befehle löschen
	globalCommands, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der globalen Befehle: %v", err)
	}
	for _, cmd := range globalCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
		if err != nil {
			log.Printf("Fehler beim Löschen des globalen Befehls %s: %v", cmd.Name, err)
		}
	}

	// Alte server-spezifische Befehle löschen
	guildCommands, err := dg.ApplicationCommands(dg.State.User.ID, "1181238521734901770")
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der server-spezifischen Befehle: %v", err)
	}
	for _, cmd := range guildCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "1181238521734901770", cmd.ID)
		if err != nil {
			log.Printf("Fehler beim Löschen des server-spezifischen Befehls %s: %v", cmd.Name, err)
		}
	}


	// Slash-Befehle registrieren
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "moneyall",
		Description: "Fügt allen Benutzern Spielgeld hinzu",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "betrag",
				Description: "Betrag an Spielgeld",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /moneyall: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "moneygive",
		Description: "Fügt einem Benutzer Spielgeld hinzu",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "benutzer",
				Description: "Der Benutzer, dem Spielgeld hinzugefügt wird",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "betrag",
				Description: "Betrag an Spielgeld",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /moneygive: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "slot",
		Description: "Spiele an der Slotmachine",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "einsatz",
				Description: "Einsatz für die Slotmachine",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /slot: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "money",
		Description: "Zeigt dir dein aktuelles Spielgeld an",
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /money: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "leaderboard",
		Description: "Zeigt die Rangliste der Spieler mit dem meisten Spielgeld an",
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /leaderboard: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "Spiele eine Runde Blackjack",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "einsatz",
				Description: "Der Betrag, den du setzen möchtest",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren des /blackjack-Befehls: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "autoslot",
		Description: "Spiele 10 Runden Slot-Maschine mit deinem Einsatz",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "bet",
				Description: "Der Einsatz für jede Runde",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren des Commands: %v", err)
	}

	handler.StartLectureTimer(dg)
	handler.StartProgressUpdater(dg)

	log.Println("Bot läuft. Drücke STRG+C zum Beenden.")

	// Auf Signal zum Beenden warten
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("Bot wird gestoppt.")
	dg.Close()
}