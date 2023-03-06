package model

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syncer/src/utils/build_info"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	l "syncer/src/utils/logger"
	"syncer/src/utils/model/sql_migrations"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewConnection(ctx context.Context, config *config.Config, applicationName string) (self *gorm.DB, err error) {
	log := l.NewSublogger("db")

	logger := logger.New(log,
		logger.Config{
			SlowThreshold:             500 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Error,           // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,                  // Disable color
		},
	)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s application_name=%s/warp.cc/%s",
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBPassword,
		config.DBName,
		config.DBSslMode,
		applicationName,
		build_info.Version,
	)

	if config.DbClientKey != "" && config.DbClientCert != "" && config.DbCaCert != "" {
		var keyFile, certFile, caFile *os.File
		keyFile, err = os.CreateTemp("", "key.pem")
		if err != nil {
			return
		}
		defer os.Remove(keyFile.Name())
		_, err = keyFile.WriteString(config.DbClientKey)
		if err != nil {
			return
		}

		certFile, err = os.CreateTemp("", "cert.pem")
		if err != nil {
			return
		}
		defer os.Remove(certFile.Name())
		_, err = certFile.WriteString(config.DbClientCert)
		if err != nil {
			return
		}

		caFile, err = os.CreateTemp("", "ca.pem")
		if err != nil {
			return
		}
		defer os.Remove(caFile.Name())
		_, err = caFile.WriteString(config.DbCaCert)
		if err != nil {
			return
		}

		dsn += fmt.Sprintf(" sslcert=%s sslkey=%s sslrootcert=%s", certFile.Name(), keyFile.Name(), caFile.Name())
	}

	self, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})
	if err != nil {
		return
	}

	err = Ping(ctx, self)
	if err != nil {
		return
	}

	// Run migrations
	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(sql_migrations.FS),
	}

	db, err := self.DB()
	if err != nil {
		return
	}

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return
	}

	log.WithField("num", n).Info("Applied migrations")

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
