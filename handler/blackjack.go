package handler

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

type BlackjackGame struct {
	UserID      string
	Deck        []string
	PlayerHand  []string
	DealerHand  []string
	Bet         int
	ChannelID   string
	MessageID   string
	GameOver    bool
}

var activeBlackjackGames = make(map[string]*BlackjackGame)

// Karten-Deck
var cards = []string{
	"A♠", "2♠", "3♠", "4♠", "5♠", "6♠", "7♠", "8♠", "9♠", "10♠", "J♠", "Q♠", "K♠",
	"A♣", "2♣", "3♣", "4♣", "5♣", "6♣", "7♣", "8♣", "9♣", "10♣", "J♣", "Q♣", "K♣",
	"A♦", "2♦", "3♦", "4♦", "5♦", "6♦", "7♦", "8♦", "9♦", "10♦", "J♦", "Q♦", "K♦",
	"A♥", "2♥", "3♥", "4♥", "5♥", "6♥", "7♥", "8♥", "9♥", "10♥", "J♥", "Q♥", "K♥",
}

func BlackjackCommand(s *discordgo.Session, m *discordgo.InteractionCreate, db *sql.DB, bet int) {
	userID := m.Member.User.ID

	// Überprüfen, ob der Benutzer bereits spielt
	if _, exists := activeBlackjackGames[userID]; exists {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du spielst bereits ein Spiel. Beende dein aktuelles Spiel zuerst.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Einsatz überprüfen
	if bet <= 0 {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Der Einsatz muss größer als 0 sein.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = ?", userID).Scan(&balance)
	if err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Abrufen deines Kontostands.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if bet > balance {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du hast nicht genug Guthaben für diesen Einsatz.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Einsatz abziehen
	_, err = db.Exec("UPDATE users SET balance = balance - ? WHERE user_id = ?", bet, userID)
	if err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Abziehen des Einsatzes.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Neues Deck mischen
	rand.Seed(time.Now().UnixNano())
	deck := make([]string, len(cards))
	copy(deck, cards)
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	// Hände initialisieren
	playerHand := []string{drawCard(&deck), drawCard(&deck)}
	dealerHand := []string{drawCard(&deck)}

	// Spiel erstellen
	game := &BlackjackGame{
		UserID:     userID,
		Deck:       deck,
		PlayerHand: playerHand,
		DealerHand: dealerHand,
		Bet:        bet,
		ChannelID:  m.ChannelID,
	}
	activeBlackjackGames[userID] = game

	// Erstelle das erste Embed
	embed := &discordgo.MessageEmbed{
		Title:       "Blackjack",
		Description: fmt.Sprintf("**Deine Karten:** %s\n**Dealer:** %s, ?\n\nWähle eine Aktion:", formatHand(playerHand), dealerHand[0]),
		Color:       0x00ff00,
	}

	// Buttons erstellen
	buttons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Hit",
					Style:    discordgo.PrimaryButton,
					CustomID: "blackjack_hit",
				},
				discordgo.Button{
					Label:    "Stay",
					Style:    discordgo.SecondaryButton,
					CustomID: "blackjack_stay",
				},
				discordgo.Button{
					Label:    "Double",
					Style:    discordgo.SuccessButton,
					CustomID: "blackjack_double",
				},
				discordgo.Button{
					Label:    "Split",
					Style:    discordgo.DangerButton,
					CustomID: "blackjack_split",
				},
			},
		},
	}

	// Antwort senden
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: buttons,
		},
	}
	err = s.InteractionRespond(m.Interaction, resp)
	if err != nil {
		delete(activeBlackjackGames, userID) // Spiel entfernen, falls etwas fehlschlägt
		return
	}

	// Nachricht speichern
	game.MessageID = m.ID
}

// Funktion zum Ziehen einer Karte
func drawCard(deck *[]string) string {
	card := (*deck)[0]
	*deck = (*deck)[1:]
	return card
}

// Hilfsfunktion zur Formatierung der Hand
func formatHand(hand []string) string {
	return fmt.Sprintf("%s", hand)
}
