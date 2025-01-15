package handler

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

var activePlayers = make(map[string]bool) // Speichert den Status, ob ein Benutzer spielt

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

// Neue Berechnung der Auszahlung basierend auf den Linien und Kombinationen
func calculatePayoutWithCombinations(board [3][3]string, bet int) (float32, []string) {
    var payout float32 = 0
    winningLinesMap := make(map[string]bool) // Verhindert Duplikate

    for _, line := range lines {
        var symbols []string

        for _, pos := range line {
            symbols = append(symbols, board[pos[0]][pos[1]])
        }

        formattedLine := strings.Join(symbols, "")
        if factor, exists := payoutFactors[formattedLine]; exists {
            if !winningLinesMap[formattedLine] { // Prüfe, ob die Linie schon existiert
                payout += float32(bet) * factor
                winningLinesMap[formattedLine] = true
            }
        }
    }

    // Konvertiere die Map in ein Array
    var winningLines []string
    for line := range winningLinesMap {
        winningLines = append(winningLines, line)
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
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ? AND guild_id = ?)", userID, guildID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("fehler beim Überprüfen der Datenbank: %v", err)
		}

		if exists {
			// Wenn der Benutzer existiert, füge den Betrag hinzu
			_, err := db.Exec("UPDATE users SET balance = ? WHERE user_id = ? AND guild_id = ?", amount, userID, guildID)
			if err != nil {
				return fmt.Errorf("fehler beim Aktualisieren des Guthabens: %v", err)
			}
		} else {
			// Wenn der Benutzer nicht existiert, füge ihn hinzu
			_, err := db.Exec("INSERT INTO users (user_id, guild_id, balance) VALUES (?, ?, ?)", userID, guildID, amount)
			if err != nil {
				return fmt.Errorf("fehler beim Hinzufügen eines neuen Benutzers: %v", err)
			}
		}
	}

	return nil
}

func MoneyGive(db *sql.DB, userID string, amount int) error {
	_, err := db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", amount, userID)
	return err
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
	
	// Balance Überprüfen
	var balance float32
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = ?", m.Member.User.ID).Scan(&balance)
	if err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler: Benutzer konnte nicht gefunden werden.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
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
	for i := 1; i <= 7; i++ {
		board = spinSlotMachine()
		embed.Description = fmt.Sprintf("%s spielt gerade!\n\n%s", fmt.Sprintf("<@%s>", m.Member.User.ID), formatSlotBoard(board))
		s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, embed)
		time.Sleep(1 * time.Second)
}

	// Gewinn berechnen
	fixedBoard := convertToFixedArray(board)
	payout, winningLines := calculatePayoutWithCombinations(fixedBoard, bet)
	if payout > 0 {
		db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", payout-float32(bet), m.Member.User.ID)
	} else {
		db.Exec("UPDATE users SET balance = balance - ? WHERE user_id = ?", bet, m.Member.User.ID)
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

func GetUserBalance(db *sql.DB, userID string, guildID string) (int, error) {
	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = ? AND guild_id = ?", userID, guildID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return balance, err
}
