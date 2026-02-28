package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gurkunwar/dailybot/internal/api"
	"github.com/Gurkunwar/dailybot/internal/bot"
	"github.com/Gurkunwar/dailybot/internal/database"
	"github.com/Gurkunwar/dailybot/internal/services"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Database: %v", err)
	}

	rdb, err := store.InitRedis()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	dg, err := bot.NewSession()
	if err != nil {
		log.Fatalf("❌ Failed to create Discord session: %v", err)
	}

	standupSvc := &services.StandupService{
		DB:      db,
		Session: dg,
	}
	userSvc := &services.UserService{DB: db}

	handler := bot.NewBotHandler(dg, rdb, db, standupSvc, userSvc)

	standupSvc.TriggerFunc = handler.Standups.InitiateStandup

	dg.AddHandler(handler.OnInteraction)

	standupSvc.StartTimezoneWorker()

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	bot.RegisterCommands(dg)

	apiServer := api.NewServer(db, dg, standupSvc)
	go apiServer.Start(":8080")

	log.Println("DailyBot is live!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
