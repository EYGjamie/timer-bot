package handler

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
)


func BlackjackHit(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB) {
	userID := i.Member.User.ID

	// Überprüfen, ob der Benutzer ein aktives Spiel hat
	game, exists := activeBlackjackGames[userID]
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du hast kein aktives Blackjack-Spiel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ziehe eine neue Karte für den Spieler
	newCard := drawCard(&game.Deck)
	game.PlayerHand = append(game.PlayerHand, newCard)

	// Berechne den Wert der Hand
	playerTotal := calculateHandValue(game.PlayerHand)

	// Beschreibung für das aktualisierte Embed
	description := fmt.Sprintf("**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s, ██",
		formatHand(game.PlayerHand), playerTotal, game.DealerHand[0])

	// Überprüfung, ob der Spieler sich überkauft hat
	if playerTotal > 21 {
		// Spieler hat sich überkauft (Bust)
		description += "\n\n**Du hast dich überkauft!** Du verlierst deinen Einsatz."
		game.GameOver = true

		// Entferne Buttons und aktualisiere das Embed
		embed := []*discordgo.MessageEmbed{
			{
				Title:       "Blackjack - Spiel beendet",
				Description: description,
				Color:       0xff0000, // Rot für Verlust
			},
		}

		// Bearbeite die Antwort oder sende eine neue Nachricht bei Fehler
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embed,
			Components: nil,
		})
		if err != nil {
			// Fallback: Überprüfe RESTError und reagiere entsprechend
			if restErr, ok := err.(*discordgo.RESTError); ok {
				if restErr.Message == nil || restErr.Message.Code == 10015 {
					// Webhook nicht verfügbar oder keine Nachricht vorhanden
					s.ChannelMessageSend(i.ChannelID, description)
				} else {
					fmt.Printf("RESTError: Code=%d, Message=%s\n", restErr.Response.StatusCode, restErr.Message)
				}
			} else {
				// Allgemeiner Fehler
				fmt.Println("Unbekannter Fehler beim Bearbeiten der Antwort:", err)
			}
		}

		// Entferne das Spiel
		delete(activeBlackjackGames, userID)
		return
	}

	// Spieler hat nicht verloren, aktualisiere das Embed
	embed := []*discordgo.MessageEmbed{
		{
			Title:       "Blackjack",
			Description: description,
			Color:       0x00ff00, // Grün für Fortsetzung
		},
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embed,
	})
	if err != nil {
		// Fallback: Überprüfe RESTError und reagiere entsprechend
		if restErr, ok := err.(*discordgo.RESTError); ok {
			if restErr.Message == nil || restErr.Message.Code == 10015 {
				// Webhook nicht verfügbar oder keine Nachricht vorhanden
				s.ChannelMessageSend(i.ChannelID, description)
			} else {
				fmt.Printf("RESTError: Code=%d, Message=%s\n", restErr.Response.StatusCode, restErr.Message)
			}
		} else {
			// Allgemeiner Fehler
			fmt.Println("Unbekannter Fehler beim Bearbeiten der Antwort:", err)
		}
	}
}


func BlackjackStay(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB) {
	userID := i.Member.User.ID

	// Überprüfen, ob der Benutzer ein aktives Spiel hat
	game, exists := activeBlackjackGames[userID]
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du hast kein aktives Blackjack-Spiel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Dealer zieht Karten, bis der Wert mindestens 17 ist
	for calculateHandValue(game.DealerHand) < 17 {
		newCard := drawCard(&game.Deck)
		game.DealerHand = append(game.DealerHand, newCard)
	}

	// Berechne die Handwerte
	playerTotal := calculateHandValue(game.PlayerHand)
	dealerTotal := calculateHandValue(game.DealerHand)

	// Ergebnislogik
	var description string
	if dealerTotal > 21 || playerTotal > dealerTotal {
		// Spieler gewinnt
		winAmount := float32(game.Bet) * 2
		_, _ = db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", winAmount, userID)
		description = fmt.Sprintf("**Du gewinnst!**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDu erhältst %.2f zurück.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal, winAmount)
	} else if dealerTotal == playerTotal {
		// Unentschieden
		_, _ = db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", float32(game.Bet), userID)
		description = fmt.Sprintf("**Unentschieden!**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDein Einsatz wurde zurückerstattet.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal)
	} else {
		// Dealer gewinnt
		description = fmt.Sprintf("**Der Dealer gewinnt.**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDu verlierst deinen Einsatz.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal)
	}

	// Entferne Buttons und aktualisiere das Embed
	embed := []*discordgo.MessageEmbed{
		{
			Title:       "Blackjack - Spiel beendet",
			Description: description,
			Color:       0xffcc00, // Gelb für Abschluss
		},
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embed, // Pointer auf das Embed-Slice
		Components: nil,    // Buttons entfernen
	})
	if err != nil {
		fmt.Println("Fehler beim Aktualisieren des Embeds:", err)
	}

	// Entferne das Spiel aus der Liste aktiver Spiele
	delete(activeBlackjackGames, userID)
}

func BlackjackDouble(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB) {
	userID := i.Member.User.ID

	// Überprüfen, ob der Benutzer ein aktives Spiel hat
	game, exists := activeBlackjackGames[userID]
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du hast kein aktives Blackjack-Spiel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Überprüfen, ob der Spieler genügend Guthaben hat, um zu verdoppeln
	var balance float32
	err := db.QueryRow("SELECT balance FROM users WHERE user_id = ?", userID).Scan(&balance)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Abrufen deines Kontostands.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if float32(game.Bet) > balance {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Du hast nicht genug Guthaben, um den Einsatz zu verdoppeln.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Einsatz verdoppeln
	game.Bet *= 2
	_, err = db.Exec("UPDATE users SET balance = balance - ? WHERE user_id = ?", game.Bet/2, userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fehler beim Aktualisieren deines Guthabens.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ziehe eine Karte für den Spieler
	newCard := drawCard(&game.Deck)
	game.PlayerHand = append(game.PlayerHand, newCard)

	// Berechne den Wert der Hand
	playerTotal := calculateHandValue(game.PlayerHand)

	// Dealer zieht Karten, bis der Wert mindestens 17 ist
	for calculateHandValue(game.DealerHand) < 17 {
		newCard := drawCard(&game.Deck)
		game.DealerHand = append(game.DealerHand, newCard)
	}

	// Berechne die Handwerte
	dealerTotal := calculateHandValue(game.DealerHand)

	// Ergebnislogik
	var description string
	if playerTotal > 21 {
		// Spieler hat sich überkauft (Bust)
		description = fmt.Sprintf("**Du hast dich überkauft!**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDu verlierst deinen Einsatz.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal)
	} else if dealerTotal > 21 || playerTotal > dealerTotal {
		// Spieler gewinnt
		winAmount := float32(game.Bet) * 2
		_, _ = db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", winAmount, userID)
		description = fmt.Sprintf("**Du gewinnst!**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDu erhältst %.2f zurück.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal, winAmount)
	} else if dealerTotal == playerTotal {
		// Unentschieden
		_, _ = db.Exec("UPDATE users SET balance = balance + ? WHERE user_id = ?", float32(game.Bet), userID)
		description = fmt.Sprintf("**Unentschieden!**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDein Einsatz wurde zurückerstattet.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal)
	} else {
		// Dealer gewinnt
		description = fmt.Sprintf("**Der Dealer gewinnt.**\n\n**Deine Karten:** %s (Wert: %d)\n**Dealer:** %s (Wert: %d)\n\nDu verlierst deinen Einsatz.",
			formatHand(game.PlayerHand), playerTotal, formatHand(game.DealerHand), dealerTotal)
	}

	// Entferne Buttons und aktualisiere das Embed
	embed := []*discordgo.MessageEmbed{
		{
			Title:       "Blackjack - Spiel beendet",
			Description: description,
			Color:       0xffcc00, // Gelb für Abschluss
		},
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embed, // Pointer auf das Embed-Slice
		Components: nil,    // Buttons entfernen
	})
	if err != nil {
		fmt.Println("Fehler beim Aktualisieren des Embeds:", err)
	}

	// Entferne das Spiel aus der Liste aktiver Spiele
	delete(activeBlackjackGames, userID)
}