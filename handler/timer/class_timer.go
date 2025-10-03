package timer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/arran4/golang-ical"
	"github.com/bwmarrin/discordgo"
)

const lectureChannelID = "1236999352329965608"
const totalLectureMinutes = 195
const icalURL = "https://stuv.app/MGH-TINF23/ical"
const maxLectureDuration = 4 * time.Hour

type LectureSlot string

const (
	Morning   LectureSlot = "morning"
	Afternoon LectureSlot = "afternoon"
	None      LectureSlot = "none"
)

type LectureEvent struct {
	Name  string
	Start time.Time
	End   time.Time
}

type ActiveLectureState struct {
	MessageID    string
	LectureSlot  LectureSlot
	Date         string
	LectureName  string
	LectureStart time.Time
	LectureEnd   time.Time
}

var currentLecture *ActiveLectureState
var cachedCalendar *ics.Calendar
var lastCalendarFetch time.Time

func fetchCalendar() (*ics.Calendar, error) {
	// Cache für 1 Woche
	if cachedCalendar != nil && time.Since(lastCalendarFetch) < 7*24*time.Hour {
		return cachedCalendar, nil
	}

	fmt.Println("📅 Lade Kalender von", icalURL)
	resp, err := http.Get(icalURL)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Abrufen des Kalenders: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Lesen des Kalenders: %w", err)
	}

	cal, err := ics.ParseCalendar(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("fehler beim Parsen des Kalenders: %w", err)
	}

	cachedCalendar = cal
	lastCalendarFetch = time.Now()

	// Zeige die nächsten 7 Tage an Vorlesungen
	printUpcomingLectures(cal, 7)

	return cal, nil
}

func convertToLocalTime(t time.Time) time.Time {
	// Zeitzone auf Europe/Berlin setzen
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		fmt.Println("Warnung: Konnte Zeitzone nicht laden:", err)
		return t
	}
	return t.In(loc)
}

func printUpcomingLectures(cal *ics.Calendar, days int) {
	fmt.Println("\n📚 Vorlesungen der nächsten", days, "Tage:")
	fmt.Println("=====================================")

	now := time.Now()
	endDate := now.AddDate(0, 0, days)

	var lectures []LectureEvent

	for _, event := range cal.Events() {
		start, err := event.GetStartAt()
		if err != nil {
			continue
		}

		end, err := event.GetEndAt()
		if err != nil {
			continue
		}

		// In lokale Zeitzone konvertieren
		start = convertToLocalTime(start)
		end = convertToLocalTime(end)

		duration := end.Sub(start)

		// Nur Events unter 4 Stunden (Vorlesungen)
		if duration >= maxLectureDuration {
			continue
		}

		// Nur Events in den nächsten X Tagen
		if start.After(now) && start.Before(endDate) {
			name := event.GetProperty(ics.ComponentPropertySummary).Value
			lectures = append(lectures, LectureEvent{
				Name:  name,
				Start: start,
				End:   end,
			})
		}
	}

	if len(lectures) == 0 {
		fmt.Println("❌ Keine Vorlesungen gefunden")
	} else {
		for _, lecture := range lectures {
			duration := lecture.End.Sub(lecture.Start)
			fmt.Printf("📖 %s\n", lecture.Name)
			fmt.Printf("   🕐 %s - %s (%s, %.0f Min.)\n",
				lecture.Start.Format("02.01.2006 15:04"),
				lecture.End.Format("15:04"),
				lecture.Start.Weekday(),
				duration.Minutes())
			fmt.Println()
		}
	}
	fmt.Println("=====================================\n")
}

func getCurrentLectureFromCalendar() *LectureEvent {
	cal, err := fetchCalendar()
	if err != nil {
		fmt.Println("Fehler beim Abrufen des Kalenders:", err)
		return nil
	}

	now := time.Now()

	for _, event := range cal.Events() {
		start, err := event.GetStartAt()
		if err != nil {
			continue
		}

		end, err := event.GetEndAt()
		if err != nil {
			continue
		}

		// In lokale Zeitzone konvertieren
		start = convertToLocalTime(start)
		end = convertToLocalTime(end)

		duration := end.Sub(start)

		// Nur Events unter 4 Stunden (Vorlesungen)
		if duration >= maxLectureDuration {
			continue
		}

		// Prüfen ob das Event gerade läuft
		if now.After(start) && now.Before(end) {
			name := event.GetProperty(ics.ComponentPropertySummary).Value
			return &LectureEvent{
				Name:  name,
				Start: start,
				End:   end,
			}
		}
	}

	return nil
}

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

func getLectureProgressFromEvent(lecture *LectureEvent) (remaining int, percentage float64) {
	now := time.Now()

	totalDuration := lecture.End.Sub(lecture.Start)
	elapsed := now.Sub(lecture.Start)
	remainingDuration := lecture.End.Sub(now)

	remaining = int(remainingDuration.Minutes())
	percentage = (elapsed.Seconds() / totalDuration.Seconds()) * 100

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

func createOrUpdateLectureEmbed(s *discordgo.Session, lecture *LectureEvent) {
	// Channel abrufen
	channel, err := s.Channel(lectureChannelID)
	if err != nil {
		fmt.Printf("Channel %s nicht gefunden: %v\n", lectureChannelID, err)
		return
	}

	remaining, percentage := getLectureProgressFromEvent(lecture)
	if remaining <= 0 {
		currentLecture = nil
		return
	}

	// Titel mit Vorlesungsname und Zeitraum
	title := fmt.Sprintf("%s", lecture.Name)
	timeRange := fmt.Sprintf("%s - %s",
		lecture.Start.Format("15:04"),
		lecture.End.Format("15:04"))

	description := "Die Vorlesung läuft noch... Durchhalten!"
	if remaining <= 0 {
		description = "Geschafft! 🎉"
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Uhrzeit",
			Value:  timeRange,
			Inline: true,
		},
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
			MessageID:    msg.ID,
			LectureSlot:  getCurrentLectureSlot(),
			Date:         time.Now().Format("2006-01-02"),
			LectureName:  lecture.Name,
			LectureStart: lecture.Start,
			LectureEnd:   lecture.End,
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
	lecture := getCurrentLectureFromCalendar()

	if lecture == nil {
		if currentLecture != nil {
			currentLecture = nil
		}
		return
	}

	// Neue Vorlesung oder keine aktive Vorlesung
	if currentLecture == nil || currentLecture.LectureName != lecture.Name || !currentLecture.LectureStart.Equal(lecture.Start) {
		currentLecture = nil
		createOrUpdateLectureEmbed(s, lecture)
	} else {
		createOrUpdateLectureEmbed(s, lecture)
	}
}

func StartLectureTimer(s *discordgo.Session) {
	fmt.Println("Starte LectureTimer-Intervall (jede Minute)")

	// Kalender beim Start einmal laden
	_, _ = fetchCalendar()

	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			CheckAndUpdateLecture(s)
		}
	}()
}

// TestCalendarDownload ist eine öffentliche Funktion zum Testen des Kalender-Downloads
func TestCalendarDownload() {
	cal, err := fetchCalendar()
	if err != nil {
		fmt.Println("❌ Fehler:", err)
		return
	}
	fmt.Println("✅ Kalender erfolgreich geladen! Anzahl Events:", len(cal.Events()))
}
