package model

import (
	"context"
	"fmt"
	"syncer/src/utils/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(ctx context.Context, config *config.Config) (self *gorm.DB, err error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBPassword,
		config.DBName,
		config.DBSSLMode)

	self, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return
	}

	sqlDB, err := self.DB()
	if err != nil {
		return
	}

	if config.DBPingTimeout > 0 {
		// Ping enabled
		dbCtx, cancel := context.WithTimeout(ctx, config.DBPingTimeout)
		defer cancel()

		err = sqlDB.PingContext(dbCtx)
		if err != nil {
			return
		}
	}

	return
}
