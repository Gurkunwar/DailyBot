package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gurkunwar/dailybot/internal/bot"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file", err)
    }

	dsn := os.Getenv("DB_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	db.AutoMigrate(&models.Guild{}, &models.UserProfile{})

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	handler := &bot.BotHanlder{
		Session: dg,
		Redis: rdb,
		DB: db,
	}

	dg.AddHandler(handler.OnMessage)
	dg.AddHandler(handler.OnInteraction)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentDirectMessages | discordgo.IntentMessageContent

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	log.Println("DailyBot is live!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}