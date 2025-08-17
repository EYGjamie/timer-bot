package slots

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	lines = [][][2]int{
		{{0, 0}, {0, 1}, {0, 2}}, // Horizontal oben
		{{1, 0}, {1, 1}, {1, 2}}, // Horizontal Mitte
		{{2, 0}, {2, 1}, {2, 2}}, // Horizontal unten
		{{0, 0}, {1, 0}, {2, 0}}, // Vertikal links
		{{0, 1}, {1, 1}, {2, 1}}, // Vertikal Mitte
		{{0, 2}, {1, 2}, {2, 2}}, // Vertikal rechts
		{{0, 0}, {1, 1}, {2, 2}}, // Diagonal \\
		{{0, 2}, {1, 1}, {2, 0}}, // Diagonal /
	}
)

var activePlayers = make(map[string]bool)

func isUserPlaying(userID string) bool {
	// Überprüft, ob der Benutzer gerade spielt
	playing, exists := activePlayers[userID]
	return exists && playing
}

func setUserPlaying(userID string, playing bool) {
	// Setzt den Spielstatus eines Benutzers
	if playing {
		activePlayers[userID] = true
	} else {
		delete(activePlayers, userID)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func getRandomSymbol() string {
	total := 0
	for _, freq := range symbolFrequencies {
		total += freq
	}

	rnd := rand.Intn(total)
	cumulative := 0
	for i, freq := range symbolFrequencies {
		cumulative += freq
		if rnd < cumulative {
			return symbols[i]
		}
	}

	return symbols[len(symbols)-1]
}

// Initialisiere das leere Slot-Board
func initializeSlotBoard() [][]string {
	return [][]string{
		{"❓", "❓", "❓"},
		{"❓", "❓", "❓"},
		{"❓", "❓", "❓"},
	}
}

// Simulation einer einzelnen Slot-Maschine-Drehung
func spinSlotMachine() [][]string {
	newBoard := make([][]string, 3) // Neues Board erstellen
	for i := 0; i < 3; i++ {
		newBoard[i] = make([]string, 3)
		for j := 0; j < 3; j++ {
			newBoard[i][j] = getRandomSymbol() // Jedes Symbol neu generieren
		}
	}
	return newBoard
}

// Slot-Board als String formatieren
func formatSlotBoard(board [][]string) string {
	lines := ""
	for _, row := range board {
		lines += fmt.Sprintf("%s | %s | %s\n", row[0], row[1], row[2])
	}
	return lines
}

// Gewinnlinien formatieren
func formatWinningLines(lines []string) string {
	if len(lines) == 0 {
		return "Keine"
	}
	return strings.Join(lines, ", ")
}

func convertToFixedArray(board [][]string) [3][3]string {
	var fixedBoard [3][3]string
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fixedBoard[i][j] = board[i][j]
		}
	}
	return fixedBoard
}

func calculatePayoutWithCombinations(board [3][3]string, bet int) (float32, []string) {
    var payout float32 = 0
    winningLinesMap := make(map[string]bool)
    var winningLines []string

    for _, line := range lines {
        var symbols []string
        var lineKeyParts []string

        for _, pos := range line {
            symbols = append(symbols, board[pos[0]][pos[1]])
            lineKeyParts = append(lineKeyParts, fmt.Sprintf("%d-%d", pos[0], pos[1]))
        }

        lineKey := strings.Join(lineKeyParts, ",")
        formattedLine := strings.Join(symbols, "")

        if factor, exists := payoutFactors[formattedLine]; exists {
            if !winningLinesMap[lineKey] {
                payout += float32(bet) * factor
                winningLinesMap[lineKey] = true
                winningLines = append(winningLines, formattedLine)
            }
        }
    }

    return payout, winningLines
}

func MoneyAll(s *discordgo.Session, db *sql.DB, guildID string, amount int) error {
	// Alle Mitglieder der Gilde abfragen
	members, err := s.GuildMembers(guildID, "", 1000)
	if err != nil {
		return fmt.Errorf("fehler beim Abrufen der Gildenmitglieder: %v", err)
	}

	for _, member := range members {
		userID := member.User.ID

		// Überprüfen, ob der Benutzer bereits in der Datenbank existiert
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND guild_id = $2)", userID, guildID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("fehler beim Überprüfen der Datenbank: %v", err)
		}

		if exists {
			// Wenn der Benutzer existiert, setze den Betrag (nicht addieren)
			_, err := db.Exec("UPDATE users SET balance = $1 WHERE user_id = $2 AND guild_id = $3", amount, userID, guildID)
			if err != nil {
				return fmt.Errorf("fehler beim Aktualisieren des Guthabens: %v", err)
			}
		} else {
			// Wenn der Benutzer nicht existiert, füge ihn hinzu
			_, err := db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES ($1, $2, $3)", userID, guildID, amount)
			if err != nil {
				return fmt.Errorf("fehler beim Hinzufügen eines neuen Benutzers: %v", err)
			}
		}
	}

	return nil
}

func MoneyGive(db *sql.DB, userID string, guildID string, amount int) error {
	// Erst prüfen, ob der Benutzer existiert
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND guild_id = $2)", userID, guildID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("fehler beim Überprüfen der Datenbank: %v", err)
	}

	if exists {
		// Benutzer existiert, Guthaben aktualisieren
		_, err = db.Exec("UPDATE users SET balance = balance + $1 WHERE user_id = $2 AND guild_id = $3", amount, userID, guildID)
		return err
	} else {
		// Benutzer existiert nicht, neuen Benutzer mit Startguthaben + Betrag erstellen
		_, err = db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES ($1, $2, $3)", userID, guildID, 1000+amount)
		return err
	}
}

func SlotCommand(s *discordgo.Session, m *discordgo.InteractionCreate, db *sql.DB, bet int) {
	// Prüfen, ob der Benutzer bereits spielt
	if isUserPlaying(m.Member.User.ID) {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du spielst bereits ein Spiel! Bitte warte, bis es beendet ist.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Benutzer als spielend markieren
	setUserPlaying(m.Member.User.ID, true)
	defer setUserPlaying(m.Member.User.ID, false)
	
	// Balance Überprüfen - mit guild_id für bessere Datenkonsistenz
	var balance float32
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = $1 AND guild_id = $2", m.Member.User.ID, m.GuildID).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			// Benutzer existiert nicht, erstelle ihn mit Startguthaben
			_, err = db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES ($1, $2, $3)", m.Member.User.ID, m.GuildID, 1000)
			if err != nil {
				s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Fehler beim Erstellen des Benutzerkontos.",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			balance = 1000
		} else {
			s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Fehler: Benutzer konnte nicht gefunden werden.",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	if bet < 1 {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Der Betrag zum spielen muss mehr als 0 sein.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if m.Member.User.ID != "423480294948208661" {
		if balance < float32(bet) {
			s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Nicht genug Spielgeld.",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Du spielst mit: %d", bet),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// Initiale Slot-Maschine anzeigen
	board := initializeSlotBoard()
	embed := &discordgo.MessageEmbed{
		Title:       "Slot Machine",
		Description: fmt.Sprintf("%s spielt gerade!\n\n%s", fmt.Sprintf("<@%s>", m.Member.User.ID), formatSlotBoard(board)),
		Color:       0x00ccff,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	msg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)

	// Animation der Slot-Maschine
	for i := 1; i <= 4; i++ {
		board = spinSlotMachine()
		embed.Description = fmt.Sprintf("%s spielt gerade!\n\n%s", fmt.Sprintf("<@%s>", m.Member.User.ID), formatSlotBoard(board))
		s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, embed)
		time.Sleep(1 * time.Second)
	}

	// Gewinn berechnen
	fixedBoard := convertToFixedArray(board)
	payout, winningLines := calculatePayoutWithCombinations(fixedBoard, bet)
	if payout > 0 {
		db.Exec("UPDATE users SET balance = balance + $1 WHERE user_id = $2 AND guild_id = $3", payout-float32(bet), m.Member.User.ID, m.GuildID)
	} else {
		db.Exec("UPDATE users SET balance = balance - $1 WHERE user_id = $2 AND guild_id = $3", bet, m.Member.User.ID, m.GuildID)
	}

	// Ergebnis-Embed
	resultEmbed := &discordgo.MessageEmbed{
		Title:       "Slot Machine Ergebnis",
		Description: fmt.Sprintf("%s, hier ist dein Ergebnis:\n\n%s", fmt.Sprintf("<@%s>", m.Member.User.ID), formatSlotBoard(board)),
		Color:       0x00ccff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Einsatz",
				Value:  fmt.Sprintf("%d", bet),
				Inline: true,
			},
			{
				Name:   "Gewinn",
				Value:  fmt.Sprintf("%.0f", payout),
				Inline: true,
			},
			{
				Name:   "Gewinnlinien",
				Value:  formatWinningLines(winningLines),
				Inline: false,
			},
			{
				Name:   "Neuer Kontostand",
				Value:  fmt.Sprintf("%.0f", float32(balance)+payout-float32(bet)),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, resultEmbed)
}

func GetUserBalance(db *sql.DB, userID string, guildID string) (float64, error) {
	var balance float64
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = $1 AND guild_id = $2", userID, guildID).Scan(&balance)
	if err == sql.ErrNoRows {
		// Benutzer existiert nicht, erstelle ihn mit Startguthaben
		_, err = db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES ($1, $2, $3)", userID, guildID, 1000)
		if err != nil {
			return 0, err
		}
		return 1000, nil
	}
	return balance, err
}

func AutoSlotCommand(s *discordgo.Session, m *discordgo.InteractionCreate, db *sql.DB, bet int) {
    // Prüfen, ob der Benutzer bereits spielt
    if isUserPlaying(m.Member.User.ID) {
        s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Du spielst bereits ein Spiel! Bitte warte, bis es beendet ist.",
                Flags: discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    // Benutzer als spielend markieren
    setUserPlaying(m.Member.User.ID, true)
    defer setUserPlaying(m.Member.User.ID, false)

    // Balance überprüfen
    var balance float32
    err := db.QueryRow("SELECT balance FROM users WHERE user_id = $1 AND guild_id = $2", m.Member.User.ID, m.GuildID).Scan(&balance)
    if err != nil {
        if err == sql.ErrNoRows {
            // Benutzer existiert nicht, erstelle ihn mit Startguthaben
            _, err = db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES ($1, $2, $3)", m.Member.User.ID, m.GuildID, 1000)
            if err != nil {
                s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
                    Type: discordgo.InteractionResponseChannelMessageWithSource,
                    Data: &discordgo.InteractionResponseData{
                        Content: "Fehler beim Erstellen des Benutzerkontos.",
                        Flags: discordgo.MessageFlagsEphemeral,
                    },
                })
                return
            }
            balance = 1000
        } else {
            s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
                Type: discordgo.InteractionResponseChannelMessageWithSource,
                Data: &discordgo.InteractionResponseData{
                    Content: "Fehler: Benutzer konnte nicht gefunden werden.",
                    Flags: discordgo.MessageFlagsEphemeral,
                },
            })
            return
        }
    }

    if bet < 1 {
        s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Der Betrag zum Spielen muss mehr als 0 sein.",
                Flags: discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    if m.Member.User.ID != "423480294948208661" {
        if balance < float32(bet*10) {
            s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
                Type: discordgo.InteractionResponseChannelMessageWithSource,
                Data: &discordgo.InteractionResponseData{
                    Content: "Nicht genug Spielgeld für 10 Spiele.",
                    Flags: discordgo.MessageFlagsEphemeral,
                },
            })
            return
        }
    }

    s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: fmt.Sprintf("Du spielst 10 Spiele mit je: %d", bet),
            Flags: discordgo.MessageFlagsEphemeral,
        },
    })
    totalPayout := float32(0)
    currentBalance := balance

    embed := &discordgo.MessageEmbed{
        Title:     "Auto Slot Machine",
        Color:     0x00ccff,
        Timestamp: time.Now().Format(time.RFC3339),
    }

    msg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)

    for i := 1; i <= 10; i++ {
        // Slot-Maschine drehen
        board := spinSlotMachine()
        fixedBoard := convertToFixedArray(board)
        payout, _ := calculatePayoutWithCombinations(fixedBoard, bet)
        totalPayout += payout

        // Guthaben aktualisieren
        if payout > 0 {
            currentBalance += payout - float32(bet)
        } else {
            currentBalance -= float32(bet)
        }

        // Embed aktualisieren
        embed.Description = fmt.Sprintf(
            " <@%s> Spiel %d/10\n\n%s\n\nEinsatz: %d\nGewinn: %.0f\nAktueller Kontostand: %.0f",
        	m.Member.User.ID,
			i,
            formatSlotBoard(board),
            bet,
            payout,
            currentBalance,
        )

        s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, embed)
        time.Sleep(1 * time.Second)
    }

    // Gesamtergebnis anzeigen
    finalEmbed := &discordgo.MessageEmbed{
        Title:       "Auto Slot Machine - Ergebnis",
        Description: fmt.Sprintf("<@%s> Nach 10 Spielen:\n\nGesamteinsatz: %d\nGesamtgewinn: %.0f\nEndkontostand: %.0f", m.Member.User.ID, bet*10, totalPayout, currentBalance),
        Color:       0x00ccff,
        Timestamp:   time.Now().Format(time.RFC3339),
    }

    s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, finalEmbed)

    // Endgültiges Guthaben in der Datenbank speichern
    db.Exec("UPDATE users SET balance = $1 WHERE user_id = $2 AND guild_id = $3", currentBalance, m.Member.User.ID, m.GuildID)
}