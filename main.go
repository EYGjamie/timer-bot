package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"discord-bot-go/handler"
)

func main() {
	// .env Datei laden
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Fehler beim Laden der .env Datei:", err)
	}

	// Token aus der .env Datei lesen
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN ist nicht definiert.")
	}

	// Discord-Sitzung erstellen
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Fehler beim Erstellen der Discord-Session:", err)
	}

	handler.StartLectureTimer(dg)

	// Bot starten
	err = dg.Open()
	if err != nil {
		log.Fatal("Fehler beim Starten der Bot-Session:", err)
	}

	log.Println("Bot läuft. Drücke STRG+C zum Beenden.")

	// Auf Signal zum Beenden warten
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("Bot wird gestoppt.")
	dg.Close()
}
