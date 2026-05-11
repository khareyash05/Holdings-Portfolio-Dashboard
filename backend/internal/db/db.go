package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
