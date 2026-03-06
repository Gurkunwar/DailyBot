package database

import (
	"os"

	"github.com/Gurkunwar/asyncflow/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	db.AutoMigrate(
		&models.Guild{},
		&models.UserProfile{},
		&models.StandupHistory{},
		&models.Standup{},

		&models.Poll{},
		&models.PollOption{},
		&models.PollVote{},
	)
	return db, nil
}