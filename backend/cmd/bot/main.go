package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gurkunwar/dailybot/internal/bot"
	"github.com/Gurkunwar/dailybot/internal/database"
	"github.com/Gurkunwar/dailybot/internal/services"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	db, _ := database.InitDB()
	rdb, _ := store.InitRedis()
	dg, _ := bot.NewSession()

	handler := &bot.BotHanlder{
		Session: dg,
		Redis: rdb,
		DB: db,
	}
	dg.AddHandler(handler.OnInteraction)

	standupSvc := &services.StandupService{
		DB:          db,
        Session:     dg,
        TriggerFunc: func(s *discordgo.Session, userID string, guildID string, standupID uint) {
             handler.InitiateStandup(s, userID, guildID, "", standupID)
        },
	}

	standupSvc.StartTimezoneWorker()

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	bot.RegisterCommands(dg)

	log.Println("DailyBot is live!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}