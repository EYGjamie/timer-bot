package timer

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

const progressChannelID = "1328643078763843604"

func calculateProgress(start, end time.Time) (remaining int, percentage float64) {
	now := time.Now()
	totalDuration := end.Sub(start).Minutes()
	elapsed := now.Sub(start).Minutes()

	remaining = int((end.Sub(now)).Minutes())
	percentage = (elapsed / totalDuration) * 100

	if remaining < 0 {
		remaining = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	return
}

func sendProgressEmbeds(s *discordgo.Session) {
	// Channel leeren (Purge)
	messages, err := s.ChannelMessages(progressChannelID, 100, "", "", "")
	if err == nil {
		for _, msg := range messages {
			s.ChannelMessageDelete(progressChannelID, msg.ID)
		}
	}

	// Fortschritt gesamtes Studium
	studyStart := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	studyEnd := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
	remaining, percentage := calculateProgress(studyStart, studyEnd)

	embedStudy := &discordgo.MessageEmbed{
		Title:       "Fortschritt des gesamten Studiums",
		Description: "Der Fortschritt des gesamten Studiums im Überblick:",
		Color:       0x00ccff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Verbleibende Zeit",
				Value:  formatTimeFromMinutes(remaining),
				Inline: true,
			},
			{
				Name:   "Fortschritt",
				Value:  fmt.Sprintf("%.1f %%", percentage),
				Inline: true,
			},
			{
				Name:   "Fortschrittsbalken",
				Value:  createProgressBar(percentage, 20),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(progressChannelID, embedStudy)

	// Fortschritt des aktuellen Semesters
	semesters := []struct {
		Semester int
		Start    time.Time
		End      time.Time
	}{
		{4, time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 27, 23, 59, 59, 0, time.UTC)},
		{5, time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 20, 23, 59, 59, 0, time.UTC)},
		{6, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 14, 23, 59, 59, 0, time.UTC)},
	}

	var currentSemester *struct {
		Semester int
		Start    time.Time
		End      time.Time
	}

	now := time.Now()
	for _, semester := range semesters {
		if now.After(semester.Start) && now.Before(semester.End) {
			currentSemester = &semester
			break
		}
	}

	if currentSemester != nil {
		remaining, percentage = calculateProgress(currentSemester.Start, currentSemester.End)
		embedSemester := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Fortschritt des %d. Semesters", currentSemester.Semester),
			Description: "Der Fortschritt des aktuellen Semesters:",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Verbleibende Zeit",
					Value:  formatTimeFromMinutes(remaining),
					Inline: true,
				},
				{
					Name:   "Fortschritt",
					Value:  fmt.Sprintf("%.1f %%", percentage),
					Inline: true,
				},
				{
					Name:   "Fortschrittsbalken",
					Value:  createProgressBar(percentage, 20),
					Inline: false,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
			s.ChannelMessageSendEmbed(progressChannelID, embedSemester)
	} else {
		s.ChannelMessageSend(progressChannelID, "Kein aktives Semester gefunden.")
	}
}

func StartProgressUpdater(s *discordgo.Session) {
	fmt.Println("Starte Fortschrittsaktualisierung alle 24 Stunden.")
	sendProgressEmbeds(s)
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			sendProgressEmbeds(s)
		}
	}()
}
