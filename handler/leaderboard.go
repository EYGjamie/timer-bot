package handler

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
	rows, err := db.Query("SELECT user_id, balance FROM users WHERE guild_id = ? ORDER BY balance DESC LIMIT 50", m.GuildID)
	if err != nil {
		s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Abrufen der Rangliste.",
			},
		})
		return
	}
	defer rows.Close()

	// Daten in einer Rangliste speichern
	leaderboard := []struct {
		UserID  string
		Balance int
	}{}

	for rows.Next() {
		var userID string
		var balance int
		if err := rows.Scan(&userID, &balance); err != nil {
			s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Fehler beim Verarbeiten der Daten.",
				},
			})
			return
		}
		leaderboard = append(leaderboard, struct {
			UserID  string
			Balance int
		}{
			UserID:  userID,
			Balance: balance,
		})
	}

	// Sortieren, falls nötig (obwohl SQL bereits sortiert)
	sort.SliceStable(leaderboard, func(i, j int) bool {
		return leaderboard[i].Balance > leaderboard[j].Balance
	})

	// Rangliste formatieren
	description := ""
	for i, entry := range leaderboard {
		username := fmt.Sprintf("<@%s>", entry.UserID) // Discord-Mention
		description += fmt.Sprintf("%d. %s - %d Spielgeld\n", i+1, username, entry.Balance)
	}

	// Embed erstellen
	embed := &discordgo.MessageEmbed{
		Title:       "Leaderboard",
		Description: description,
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
