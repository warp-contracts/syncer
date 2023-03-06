package notify

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"syncer/src/utils/build_info"
	"syncer/src/utils/config"
	"syncer/src/utils/task"
	"time"

	"github.com/jackc/pgx"
)

// Streams data from postgres notification channel
// puts on output channel
type Streamer struct {
	*task.Task

	pool       *pgx.ConnPool
	connection *pgx.Conn

	channelName string

	Output chan string
}

func NewStreamer(config *config.Config, name string) (self *Streamer) {
	self = new(Streamer)

	self.Output = make(chan string)

	self.Task = task.NewTask(config, name).
		WithSubtaskFunc(self.run).
		// WithPeriodicSubtaskFunc(10*time.Second, self.monitor).
		WithOnBeforeStart(self.connect).
		WithOnStop(func() {
			close(self.Output)
		}).
		WithOnAfterStop(self.disconnect)

	return
}

func (self *Streamer) WithNotificationChannelName(name string) *Streamer {
	self.channelName = name
	return self
}

func (self *Streamer) WithCapacity(size int) *Streamer {
	self.Output = make(chan string, size)
	return self
}

func (self *Streamer) disconnect() {
	err := self.connection.Close()
	if err != nil {
		self.Log.WithError(err).Error("Failed to close connection")
	}

	self.pool.Close()
}

func (self *Streamer) connect() (err error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s application_name=%s/warp.cc/%s",
		self.Config.DBHost,
		self.Config.DBPort,
		self.Config.DBUser,
		self.Config.DBPassword,
		self.Config.DBName,
		self.Config.DBSslMode,
		self.Name,
		build_info.Version)

	config, err := pgx.ParseDSN(dsn)
	if err != nil {
		return
	}

	if self.Config.DbClientCert != "" && self.Config.DbClientKey != "" && self.Config.DbCaCert != "" {
		cert, err := tls.X509KeyPair([]byte(self.Config.DbClientCert), []byte(self.Config.DbClientKey))
		if err != nil {
			self.Log.WithError(err).Error("Failed to load client cert")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(self.Config.DbCaCert)) {
			return errors.New("failed to append CA cert to pool")
		}

		config.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
			ClientCAs:          caCertPool,
			Certificates:       []tls.Certificate{cert},
		}
	}

	self.pool, err = pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: config})
	if err != nil {
		return
	}

	self.connection, err = self.pool.Acquire()
	if err != nil {
		return
	}

	return
}

func (self *Streamer) reconnect() {
	var err error
	for {
		err = self.connect()
		if err == nil {
			// SUCCESS
			self.Log.Info("Connection established")
			return
		}

		self.Log.WithError(err).Error("Failed to connect, retrying in 1s")
		time.Sleep(time.Second)
		if self.IsStopping.Load() {
			return
		}
	}
}

func (self *Streamer) run() (err error) {
	err = self.connection.Listen(self.channelName)
	if err != nil {
		return
	}

	defer func() {
		err = self.connection.Unlisten(self.channelName)
		if err != nil {
			self.Log.WithError(err).Error("Failed to unlisten channel")
		}
	}()

	for {
		// Waits for notification unless task gets stopped
		msg, err := self.connection.WaitForNotification(self.Ctx)
		if errors.Is(err, context.Canceled) {
			// Stop() was called
			return nil
		}

		if err != nil {
			self.Log.WithError(err).Error("Failed to wait for notification, reconnecting")
			self.reconnect()
		} else {
			// Send notification to output channel
			self.Output <- msg.Payload
		}
	}
}
