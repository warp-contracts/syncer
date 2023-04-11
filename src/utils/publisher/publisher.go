package publisher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding"
	"errors"
	"fmt"
	"syncer/src/utils/config"
	"syncer/src/utils/task"
	"time"

	"github.com/redis/go-redis/v9"
)

// Forwards messages to Redis
type Publisher[In encoding.BinaryMarshaler] struct {
	*task.Task

	client      *redis.Client
	channelName string
	input       chan In
}

func NewPublisher[In encoding.BinaryMarshaler](config *config.Config, name string) (self *Publisher[In]) {
	self = new(Publisher[In])

	self.Task = task.NewTask(config, name).
		WithSubtaskFunc(self.run).
		WithOnBeforeStart(self.connect).
		WithOnAfterStop(self.disconnect)

	return
}

func (self *Publisher[In]) WithInputChannel(v chan In) *Publisher[In] {
	self.input = v
	return self
}

func (self *Publisher[In]) WithChannelName(v string) *Publisher[In] {
	self.channelName = v
	return self
}

func (self *Publisher[In]) disconnect() {
	err := self.client.Close()
	if err != nil {
		self.Log.WithError(err).Error("Failed to close connection")
	}
}

func (self *Publisher[In]) connect() (err error) {
	opts := redis.Options{
		ClientName:      fmt.Sprintf("warp.cc/%s", self.Name),
		Addr:            fmt.Sprintf("%s:%d", self.Config.Redis.Host, self.Config.Redis.Port),
		Password:        self.Config.Redis.Password,
		Username:        self.Config.Redis.User,
		DB:              self.Config.Redis.DB,
		MinIdleConns:    self.Config.Redis.MinIdleConns,
		MaxIdleConns:    self.Config.Redis.MaxIdleConns,
		ConnMaxIdleTime: self.Config.Redis.ConnMaxIdleTime,
		PoolSize:        self.Config.Redis.MaxOpenConns,
		ConnMaxLifetime: self.Config.Redis.ConnMaxLifetime,
	}

	if self.Config.Redis.ClientCert != "" && self.Config.Redis.ClientKey != "" && self.Config.Redis.CaCert != "" {
		cert, err := tls.X509KeyPair([]byte(self.Config.Database.ClientCert), []byte(self.Config.Database.ClientKey))
		if err != nil {
			self.Log.WithError(err).Error("Failed to load client cert")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(self.Config.Database.CaCert)) {
			return errors.New("failed to append CA cert to pool")
		}

		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
			ClientCAs:          caCertPool,
			Certificates:       []tls.Certificate{cert},
		}
	}

	self.client = redis.NewClient(&opts)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return self.client.Ping(ctx).Err()
}

func (self *Publisher[In]) run() (err error) {
	self.Log.Info("Starting publisher")
	for payload := range self.input {
		self.Log.Info("Payload")
		err = self.client.Publish(self.Ctx, self.channelName, payload).Err()
		if err != nil {
			self.Log.WithError(err).Error("Failed to publish message")
		}
	}
	return nil
}
