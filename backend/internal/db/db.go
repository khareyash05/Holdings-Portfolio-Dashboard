package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
)

func Connect(url string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(url), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sdb, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	if err := sdb.Ping(); err != nil {
		return nil, err
	}
	return gdb, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.Exchange{}, &models.Stock{}, &models.Holding{})
}
