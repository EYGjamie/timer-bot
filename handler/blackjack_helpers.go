package handler

import "fmt"

func drawCard(deck *[]string) string {
	card := (*deck)[0]
	*deck = (*deck)[1:]
	return card
}

func isBlackjack(hand []string) bool {
	if len(hand) != 2 {
		return false
	}
	return (hand[0][0] == 'A' && isFaceCard(hand[1])) || (hand[1][0] == 'A' && isFaceCard(hand[0]))
}

func isFaceCard(card string) bool {
	return card[0] == 'J' || card[0] == 'Q' || card[0] == 'K' || card[:2] == "10"
}

func calculateHandValue(hand []string) int {
	total := 0
	aces := 0

	for _, card := range hand {
		switch card[0] {
		case 'A':
			aces++
			total += 11
		case 'K', 'Q', 'J':
			total += 10
		default:
			var value int
			if card[:2] == "10" {
				value = 10
			} else {
				value = int(card[0] - '0')
			}
			total += value
		}
	}

	// Passe die Asse an, falls der Spieler überkauft
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}

	return total
}

func formatHand(hand []string) string {
	return fmt.Sprintf("%s", hand)
}
