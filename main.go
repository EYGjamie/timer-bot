package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"discord-bot-go/handler/timer"
	"discord-bot-go/db"
	"discord-bot-go/handler/leaderboard"
	"discord-bot-go/handler/slots"
)

func main() {
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
	connStr := database.GetPostgreSQLConnectionString()

	// Datenbankverbindung herstellen (mit Retry-Logik)
	log.Printf("Verbinde mit PostgreSQL Datenbank...")
	db, err := database.WaitForDatabase(connStr)
	if err != nil {
		log.Fatalf("Fehler bei der Datenbankverbindung: %v", err)
	}
	defer db.Close()

	// Connection Pool konfigurieren
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Datenbank initialisieren
	if err := database.InitDatabase(db); err != nil {
		log.Fatalf("Fehler bei der Datenbankinitialisierung: %v", err)
	}

	log.Println("✅ PostgreSQL Datenbank erfolgreich initialisiert!")

	// Discord-Session mit Intents erstellen
	intents := discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Fehler beim Erstellen der Discord-Session:", err)
	}
	dg.Identify.Intents = intents
	
	const ownerID = "423480294948208661"

	// Event-Handler registrieren
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		switch m.Type {
		case discordgo.InteractionApplicationCommand:
			switch m.ApplicationCommandData().Name {

			case "moneyall":
				if m.Member.User.ID != ownerID {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Du bist nicht berechtigt, diesen Befehl auszuführen.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				amount := m.ApplicationCommandData().Options[0].IntValue()
				err := slots.MoneyAll(s, db, m.GuildID, int(amount))
				if err != nil {
					log.Printf("Fehler bei MoneyAll: %v", err)
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Fehler: %v", err),
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
				} else {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Allen Mitgliedern wurde das Guthaben auf %d Müller Coins gesetzt!", amount),
						},
					})
				}

			case "moneygive":
				if m.Member.User.ID != ownerID {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Du bist nicht berechtigt, diesen Befehl auszuführen.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				// Führe den eigentlichen Befehl aus
				userID := m.ApplicationCommandData().Options[0].UserValue(nil).ID
				amount := m.ApplicationCommandData().Options[1].IntValue()
				err := slots.MoneyGive(db, userID, m.GuildID, int(amount))
				if err != nil {
					log.Printf("Fehler bei MoneyGive: %v", err)
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Fehler beim Hinzufügen von Spielgeld.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
				} else {
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%d Müller Coins wurden Benutzer <@%s> hinzugefügt.", amount, userID),
						},
					})
				}

			case "slot":
				bet := m.ApplicationCommandData().Options[0].IntValue()
				slots.SlotCommand(s, m, db, int(bet))

			case "money":
				// Aktuelles Spielgeld des Benutzers abrufen
				balance, err := slots.GetUserBalance(db, m.Member.User.ID, m.GuildID)
				if err != nil {
					log.Printf("Fehler bei GetUserBalance: %v", err)
					s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Fehler beim Abrufen deines Guthabens. Bitte versuche es später erneut.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				// Antwort mit dem Guthaben senden
				s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Du hast aktuell %.0f Müller Coins.", balance),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			case "leaderboard":
				// Aufruf des Leaderboard-Handlers
				leaderboard.LeaderboardHandler(s, m, db)
			
			case "autoslot":
				bet := m.ApplicationCommandData().Options[0].IntValue()
				slots.AutoSlotCommand(s, m, db, int(bet))

			default:
				log.Printf("Unbekannter Befehl: %s", m.ApplicationCommandData().Name)
			}

		default:
			log.Printf("Unbekannter Interaktionstyp: %v", m.Type)
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

	log.Printf("✅ Bot gestartet als: %s#%s", dg.State.User.Username, dg.State.User.Discriminator)

	// Alte globale Befehle löschen
	log.Println("Lösche alte globale Befehle...")
	globalCommands, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Printf("Warnung: Fehler beim Abrufen der globalen Befehle: %v", err)
	} else {
		for _, cmd := range globalCommands {
			err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
			if err != nil {
				log.Printf("Fehler beim Löschen des globalen Befehls %s: %v", cmd.Name, err)
			} else {
				log.Printf("Globaler Befehl gelöscht: %s", cmd.Name)
			}
		}
	}

	// Alte server-spezifische Befehle löschen
	log.Println("Lösche alte server-spezifische Befehle...")
	guildCommands, err := dg.ApplicationCommands(dg.State.User.ID, "1181238521734901770")
	if err != nil {
		log.Printf("Warnung: Fehler beim Abrufen der server-spezifischen Befehle: %v", err)
	} else {
		for _, cmd := range guildCommands {
			err := dg.ApplicationCommandDelete(dg.State.User.ID, "1181238521734901770", cmd.ID)
			if err != nil {
				log.Printf("Fehler beim Löschen des server-spezifischen Befehls %s: %v", cmd.Name, err)
			} else {
				log.Printf("Server-spezifischer Befehl gelöscht: %s", cmd.Name)
			}
		}
	}

	// Neue Slash-Befehle registrieren
	log.Println("Registriere neue Slash-Befehle...")

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "1181238521734901770", &discordgo.ApplicationCommand{
		Name:        "moneyall",
		Description: "Setzt das Guthaben aller Benutzer auf einen bestimmten Betrag",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "betrag",
				Description: "Neuer Guthabenbetrag für alle Benutzer",
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
				Description: "Betrag an Spielgeld der hinzugefügt wird",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /moneygive: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "slot",
		Description: "Spiele an der Slotmachine",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "einsatz",
				Description: "Einsatz für die Slotmachine (Mindestens 1)",
				Required:    true,
				MinValue:    &[]float64{1}[0],
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /slot: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "money",
		Description: "Zeigt dir dein aktuelles Spielgeld an",
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /money: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "leaderboard",
		Description: "Zeigt die Rangliste der Spieler mit dem meisten Spielgeld an",
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /leaderboard: %v", err)
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "autoslot",
		Description: "Spiele 10 Runden Slot-Maschine automatisch",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "einsatz",
				Description: "Der Einsatz pro Runde (Mindestens 1)",
				Required:    true,
				MinValue:    &[]float64{1}[0],
			},
		},
	})
	if err != nil {
		log.Fatalf("Fehler beim Registrieren von /autoslot: %v", err)
	}

	log.Println("✅ Alle Slash-Befehle erfolgreich registriert!")

	// Timer starten
	log.Println("Starte Timer...")
	timer.StartLectureTimer(dg)
	timer.StartProgressUpdater(dg)

	log.Println("🎉 Bot läuft erfolgreich! Drücke STRG+C zum Beenden.")

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("🛑 Bot wird gestoppt...")
	dg.Close()
	log.Println("✅ Bot erfolgreich gestoppt.")
}