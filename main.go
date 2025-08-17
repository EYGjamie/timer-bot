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
	log.Printf("Verbinde mit Postgre DB")
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
				err := slots.MoneyGive(db, userID, int(amount))
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
				slots.SlotCommand(s, m, db, int(bet))

			case "money":
				// Aktuelles Spielgeld des Benutzers abrufen
				balance, err := slots.GetUserBalance(db, m.Member.User.ID, m.GuildID)
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
				leaderboard.LeaderboardHandler(s, m, db)
			
			case "autoslot":
				bet := m.ApplicationCommandData().Options[0].IntValue()
				slots.AutoSlotCommand(s, m, db, int(bet))

			}

		default:
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

	timer.StartLectureTimer(dg)
	timer.StartProgressUpdater(dg)

	log.Println("Bot läuft. Drücke STRG+C zum Beenden.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("Bot wird gestoppt.")
	dg.Close()
}