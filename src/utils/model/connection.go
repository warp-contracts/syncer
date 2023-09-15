package model

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/warp-contracts/syncer/src/utils/build_info"
	"github.com/warp-contracts/syncer/src/utils/config"
	l "github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model/sql_migrations"

	migrate "github.com/rubenv/sql-migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(ctx context.Context, dbConfig *config.Database, username, password, applicationName string) (self *gorm.DB, err error) {
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
		dbConfig.Host,
		dbConfig.Port,
		username,
		password,
		dbConfig.Name,
		dbConfig.SslMode,
		applicationName,
		build_info.Version,
	)

	if dbConfig.CaCertPath != "" && dbConfig.ClientKeyPath != "" && dbConfig.ClientCertPath != "" {
		log.Info("Using SSL certificates from files")
		dsn += fmt.Sprintf(" sslcert=%s sslkey=%s sslrootcert=%s", dbConfig.ClientCertPath, dbConfig.ClientKeyPath, dbConfig.CaCertPath)
	} else if dbConfig.ClientKey != "" && dbConfig.ClientCert != "" && dbConfig.CaCert != "" {
		log.Info("Using SSL certificates from variables")

		var keyFile, certFile, caFile *os.File
		keyFile, err = os.CreateTemp("", "key.pem")
		if err != nil {
			return
		}
		defer os.Remove(keyFile.Name())
		_, err = keyFile.WriteString(dbConfig.ClientKey)
		if err != nil {
			return
		}

		certFile, err = os.CreateTemp("", "cert.pem")
		if err != nil {
			return
		}
		defer os.Remove(certFile.Name())
		_, err = certFile.WriteString(dbConfig.ClientCert)
		if err != nil {
			return
		}

		caFile, err = os.CreateTemp("", "ca.pem")
		if err != nil {
			return
		}
		defer os.Remove(caFile.Name())
		_, err = caFile.WriteString(dbConfig.CaCert)
		if err != nil {
			return
		}

		dsn += fmt.Sprintf(" sslcert=%s sslkey=%s sslrootcert=%s", certFile.Name(), keyFile.Name(), caFile.Name())
	}

	self, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})
	if err != nil {
		return
	}

	db, err := self.DB()
	if err != nil {
		return
	}

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxIdleTime(dbConfig.ConnMaxIdleTime)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)
	err = ping(ctx, dbConfig, self)
	if err != nil {
		return
	}

	return
}

func NewConnection(ctx context.Context, config *config.Config, applicationName string) (self *gorm.DB, err error) {
	err = Migrate(ctx, config)
	if err != nil {
		return
	}

	return Connect(ctx, &config.Database, config.Database.User, config.Database.Password, applicationName)
}

func NewReadOnlyConnection(ctx context.Context, config *config.Config, applicationName string) (self *gorm.DB, err error) {
	return Connect(ctx, &config.ReadOnlyDatabase, config.Database.User, config.Database.Password, applicationName)
}

func Migrate(ctx context.Context, config *config.Config) (err error) {
	log := l.NewSublogger("db-migrate")

	if config.Database.MigrationUser == "" || config.Database.MigrationPassword == "" {
		log.Info("Migration user not set, skipping migrations")
		return
	}

	// Run migrations
	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(sql_migrations.FS),
	}

	// Use special migration user
	self, err := Connect(ctx, &config.Database, config.Database.MigrationUser, config.Database.MigrationPassword, "migration")
	if err != nil {
		return
	}

	db, err := self.DB()
	if err != nil {
		return
	}
	defer db.Close()

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return
	}

	log.WithField("num", n).Info("Applied migrations")

	config.Database.MigrationUser = ""
	config.Database.MigrationPassword = ""

	return
}

func ping(ctx context.Context, dbConfig *config.Database, db *gorm.DB) (err error) {
	if dbConfig.PingTimeout < 0 {
		// Ping disabled
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	dbCtx, cancel := context.WithTimeout(ctx, dbConfig.PingTimeout)
	defer cancel()

	err = sqlDB.PingContext(dbCtx)
	if err != nil {
		return
	}
	return
}
