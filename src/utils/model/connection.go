package model

import (
	"context"
	"fmt"
	"syncer/src/utils/common"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewConnection(ctx context.Context) (self *gorm.DB, err error) {
	config := common.GetConfig(ctx)

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

	err = Ping(ctx, self)
	if err != nil {
		return
	}

	// Migrate state changes
	err = self.AutoMigrate(State{})
	if err != nil {
		return
	}

	// Ensure there's syncer_state has one row inserted
	self.Clauses(clause.OnConflict{
		// Columns:   cols,
		// DoUpdates: clause.AssignmentColumns(colsNames),
		DoNothing: true,
	}).
		Create(&State{
			Id: 1,
		})

	return
}

func Ping(ctx context.Context, db *gorm.DB) (err error) {
	config := common.GetConfig(ctx)

	if config.DBPingTimeout < 0 {
		// Ping disabled
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	dbCtx, cancel := context.WithTimeout(ctx, config.DBPingTimeout)
	defer cancel()

	err = sqlDB.PingContext(dbCtx)
	if err != nil {
		return
	}
	return
}
