package notify

import (
	"context"
	"errors"
	"fmt"
	"syncer/src/utils/config"
	"syncer/src/utils/task"

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

func NewStreamer(config *config.Config) (self *Streamer) {
	self = new(Streamer)

	self.Output = make(chan string)

	self.Task = task.NewTask(config, "streamer").
		WithSubtaskFunc(self.run).
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
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		self.Config.DBHost,
		self.Config.DBPort,
		self.Config.DBUser,
		self.Config.DBPassword,
		self.Config.DBName,
		self.Config.DBSslMode)

	config, err := pgx.ParseDSN(dsn)
	if err != nil {
		return
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
			self.Log.WithError(err).Error("Failed to wait for notification")
		} else {
			// Send notification to output channel
			self.Output <- msg.Payload
		}
	}
}
