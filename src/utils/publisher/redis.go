package publisher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding"
	"errors"
	"fmt"
	"syncer/src/utils/config"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"time"

	"github.com/redis/go-redis/v9"
)

// Forwards messages to Redis
type RedisPublisher[In encoding.BinaryMarshaler] struct {
	*task.Task

	redisConfig config.Redis

	monitor monitoring.Monitor

	client      *redis.Client
	channelName string
	input       chan In
}

func NewRedisPublisher[In encoding.BinaryMarshaler](config *config.Config, redisConfig config.Redis, name string) (self *RedisPublisher[In]) {
	self = new(RedisPublisher[In])

	self.redisConfig = redisConfig

	self.Task = task.NewTask(config, name).
		WithSubtaskFunc(self.run).
		WithOnBeforeStart(self.connect).
		WithOnAfterStop(self.disconnect).
		WithWorkerPool(redisConfig.MaxWorkers, redisConfig.MaxQueueSize)

	return
}

func (self *RedisPublisher[In]) WithInputChannel(v chan In) *RedisPublisher[In] {
	self.input = v
	return self
}

func (self *RedisPublisher[In]) WithChannelName(v string) *RedisPublisher[In] {
	self.channelName = v
	return self
}

func (self *RedisPublisher[In]) WithMonitor(monitor monitoring.Monitor) *RedisPublisher[In] {
	self.monitor = monitor
	return self
}

func (self *RedisPublisher[In]) disconnect() {
	err := self.client.Close()
	if err != nil {
		self.Log.WithError(err).Error("Failed to close connection")
	}
}

func (self *RedisPublisher[In]) connect() (err error) {
	opts := redis.Options{
		ClientName:      fmt.Sprintf("warp.cc/%s", self.Name),
		Addr:            fmt.Sprintf("%s:%d", self.redisConfig.Host, self.redisConfig.Port),
		Password:        self.redisConfig.Password,
		Username:        self.redisConfig.User,
		DB:              self.redisConfig.DB,
		MinIdleConns:    self.redisConfig.MinIdleConns,
		MaxIdleConns:    self.redisConfig.MaxIdleConns,
		ConnMaxIdleTime: self.redisConfig.ConnMaxIdleTime,
		PoolSize:        self.redisConfig.MaxOpenConns,
		ConnMaxLifetime: self.redisConfig.ConnMaxLifetime,
	}

	if self.redisConfig.ClientCert != "" && self.redisConfig.ClientKey != "" && self.redisConfig.CaCert != "" {
		cert, err := tls.X509KeyPair([]byte(self.redisConfig.ClientCert), []byte(self.redisConfig.ClientKey))
		if err != nil {
			self.Log.WithError(err).Error("Failed to load client cert")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(self.redisConfig.CaCert)) {
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
	err = self.client.Ping(ctx).Err()
	if err != nil {
		self.Log.WithError(err).Error("Failed to ping Redis")
		return
	}

	return
}

func (self *RedisPublisher[In]) run() (err error) {
	for payload := range self.input {

		self.SubmitToWorker(func() {
			self.Log.Debug("Redis publish...")
			defer self.Log.Debug("...Redis publish done")
			err = task.NewRetry().
				WithContext(self.Ctx).
				WithMaxElapsedTime(self.redisConfig.MaxElapsedTime).
				WithMaxInterval(self.redisConfig.MaxInterval).
				WithOnError(func(err error, isDurationAcceptable bool) error {
					self.Log.WithError(err).Error("Failed to publish message, retrying")
					self.monitor.GetReport().RedisPublisher.Errors.Publish.Inc()
					return err
				}).
				Run(func() (err error) {
					return self.client.Publish(self.Ctx, self.channelName, payload).Err()
				})
			if err != nil {
				self.Log.WithError(err).Error("Failed to publish message, giving up")
				self.monitor.GetReport().RedisPublisher.Errors.PersistentFailure.Inc()
				return
			}
			self.monitor.GetReport().RedisPublisher.State.MessagesPublished.Inc()
		})
	}
	return nil
}
