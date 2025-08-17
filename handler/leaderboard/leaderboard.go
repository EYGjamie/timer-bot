package leaderboard

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Handler für das Leaderboard
func LeaderboardHandler(s *discordgo.Session, m *discordgo.InteractionCreate, db *sql.DB) {
	// Daten aus der Datenbank abrufen, gefiltert nach der aktuellen Guild-ID
	rows, err := db.Query("SELECT user_id, balance FROM users WHERE guild_id = $1 ORDER BY balance DESC LIMIT 50", m.GuildID)
	if err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Abrufen der Rangliste.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	defer rows.Close()

	// Daten in einer Rangliste speichern
	leaderboard := []struct {
		UserID  string
		Balance float64
	}{}

	for rows.Next() {
		var userID string
		var balance float64
		if err := rows.Scan(&userID, &balance); err != nil {
			s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Fehler beim Verarbeiten der Daten.",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		leaderboard = append(leaderboard, struct {
			UserID  string
			Balance float64
		}{
			UserID:  userID,
			Balance: balance,
		})
	}

	// Prüfe auf mögliche Fehler beim Iterieren über die Rows
	if err = rows.Err(); err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Lesen der Datenbankdaten.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Prüfe ob Daten vorhanden sind
	if len(leaderboard) == 0 {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Keine Spielerdaten gefunden. Spielt zuerst ein paar Runden!",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Sortieren, falls nötig (obwohl SQL bereits sortiert)
	sort.SliceStable(leaderboard, func(i, j int) bool {
		return leaderboard[i].Balance > leaderboard[j].Balance
	})

	// Rangliste formatieren
	description := ""
	for i, entry := range leaderboard {
		position := fmt.Sprintf("%d.", i+1)
		if i == 0 {
			position = "🥇"
		} else if i == 1 {
			position = "🥈"
		} else if i == 2 {
			position = "🥉"
		}

		username := fmt.Sprintf("<@%s>", entry.UserID)
		description += fmt.Sprintf("%s %s - %.0f Müller Coins\n", position, username, entry.Balance)
	}

	// Embed erstellen
	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Leaderboard - Müller Coins",
		Description: description,
		Color:       0x00ff00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Insgesamt %d Spieler", len(leaderboard)),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}