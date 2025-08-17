package timer

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

const lectureChannelID = "1236999352329965608"
const totalLectureMinutes = 195

type LectureSlot string

const (
	Morning   LectureSlot = "morning"
	Afternoon LectureSlot = "afternoon"
	None      LectureSlot = "none"
)

type ActiveLectureState struct {
	MessageID   string
	LectureSlot LectureSlot
	Date        string
}

var currentLecture *ActiveLectureState

func getCurrentLectureSlot() LectureSlot {
	now := time.Now()
	day := now.Weekday()

	// Wochenende ausschließen
	if day == time.Saturday || day == time.Sunday {
		return None
	}

	hour := now.Hour()
	minute := now.Minute()
	totalMinutes := hour*60 + minute

	// 9:00 - 12:15
	if totalMinutes >= 540 && totalMinutes < 735 {
		return Morning
	}
	// 13:00 - 16:15
	if totalMinutes >= 780 && totalMinutes < 975 {
		return Afternoon
	}

	return None
}

func getLectureTimeRange(slot LectureSlot) (start int, end int) {
	switch slot {
	case Morning:
		return 540, 735
	case Afternoon:
		return 780, 975
	default:
		return 0, 0
	}
}

func getLectureProgress(slot LectureSlot) (remaining int, percentage float64) {
	start, end := getLectureTimeRange(slot)
	if start == 0 && end == 0 {
		return 0, 0
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	elapsed := currentMinutes - start
	remaining = end - currentMinutes
	percentage = float64(elapsed) / float64(totalLectureMinutes) * 100

	return
}

func formatTimeFromMinutes(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func createProgressBar(percentage float64, maxBars int) string {
	filledBars := int((percentage / 100) * float64(maxBars))
	emptyBars := maxBars - filledBars
	return fmt.Sprintf("[%s%s]", repeatString("█", filledBars), repeatString("░", emptyBars))
}

func repeatString(char string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result
}

func createOrUpdateLectureEmbed(s *discordgo.Session, slot LectureSlot) {
	// Channel abrufen
	channel, err := s.Channel(lectureChannelID)
	if err != nil {
		fmt.Printf("Channel %s nicht gefunden: %v\n", lectureChannelID, err)
		return
	}

	remaining, percentage := getLectureProgress(slot)
	if slot == None || remaining <= 0 {
		currentLecture = nil
		return
	}

	title := ""
	switch slot {
	case Morning:
		title = "Vorlesung (Morgen) 9:00 - 12:15"
	case Afternoon:
		title = "Vorlesung (Nachmittag) 13:00 - 16:15"
	}

	description := "Die Vorlesung läuft noch... Durchhalten!"
	if remaining <= 0 {
		description = "Geschafft! 🎉"
	}

	fields := []*discordgo.MessageEmbedField{
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
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ccff,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if currentLecture == nil {
		// Neue Nachricht erstellen
		msg, err := s.ChannelMessageSendEmbed(channel.ID, embed)
		if err != nil {
			fmt.Println("Fehler beim Senden der Nachricht:", err)
			return
		}

		currentLecture = &ActiveLectureState{
			MessageID:   msg.ID,
			LectureSlot: slot,
			Date:        time.Now().Format("2006-01-02"),
		}
	} else {
		// Nachricht aktualisieren
		_, err := s.ChannelMessageEditEmbed(channel.ID, currentLecture.MessageID, embed)
		if err != nil {
			fmt.Println("Fehler beim Bearbeiten der Nachricht:", err)
		}
	}
}

func CheckAndUpdateLecture(s *discordgo.Session) {
	slot := getCurrentLectureSlot()
	if slot == None {
		if currentLecture != nil {
			currentLecture = nil
		}
		return
	}

	if currentLecture == nil || currentLecture.LectureSlot != slot {
		currentLecture = nil
		createOrUpdateLectureEmbed(s, slot)
	} else {
		createOrUpdateLectureEmbed(s, slot)
	}
}

func StartLectureTimer(s *discordgo.Session) {
	fmt.Println("Starte LectureTimer-Intervall (jede Minute)")
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			CheckAndUpdateLecture(s)
		}
	}()
}
