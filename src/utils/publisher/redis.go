package publisher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

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
		WithPeriodicSubtaskFunc(time.Second*30, self.ping).
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
	self.Log.WithField("port", self.redisConfig.Port).WithField("host", self.redisConfig.Host).Info("Setup Redis connection")
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
		PoolFIFO:        true,
		PoolTimeout:     time.Minute,
		OnConnect: func(ctx context.Context, con *redis.Conn) error {
			self.Log.WithField("state", con.String()).WithField("host", self.redisConfig.Host).Info("Connected to Redis")
			return nil
		},
		ReadTimeout:     time.Second * 30,
		WriteTimeout:    time.Second * 30,
		DialTimeout:     time.Minute,
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
	} else if self.redisConfig.EnableTLS {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	opts.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netDialer := &net.Dialer{
			Timeout:   time.Minute,
			KeepAlive: time.Second * 30,
		}
		if opts.TLSConfig == nil {
			return nil, errors.New("TLS config is nil")
		}
		return tls.DialWithDialer(netDialer, network, addr, opts.TLSConfig)
	}

	self.client = redis.NewClient(&opts)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = self.client.Ping(ctx).Err()
	if err != nil {
		self.Log.WithError(err).Error("Failed to ping Redis")
		return
	}
	self.Log.WithField("host", self.redisConfig.Host).Info("Redis connection OK")

	return
}

func (self *RedisPublisher[In]) ping() (err error) {
	ctx, cancel := context.WithTimeout(self.Ctx, 30*time.Second)
	defer cancel()

	// No need to monitor connection with another message if messages go through
	if time.Now().Unix()-self.monitor.GetReport().RedisPublisher.State.LastSuccessfulMessageTimestamp.Load() < 30 {
		return nil
	}

	// Test the connection with a PING message
	self.Log.Debug("Monitor Redis connection")
	err = self.client.Ping(ctx).Err()
	if err != nil {
		self.Log.WithError(err).Error("Failed to ping Redis")
		return
	}
	self.monitor.GetReport().RedisPublisher.State.LastSuccessfulMessageTimestamp.Store(time.Now().Unix())

	return nil
}

func (self *RedisPublisher[In]) run() (err error) {
	i := 0
	for payload := range self.input {
		i++
		i := i

		self.Log.WithField("i", i).Debug("Redis publish...")

		err = task.NewRetry().
			WithContext(self.Ctx).
			WithMaxElapsedTime(self.redisConfig.MaxElapsedTime).
			WithMaxInterval(self.redisConfig.MaxInterval).
			WithOnError(func(err error, isDurationAcceptable bool) error {
				self.Log.WithError(err).Warn(" , retrying")
				self.monitor.GetReport().RedisPublisher.Errors.Publish.Inc()
				return err
			}).
			Run(func() (err error) {
				self.Log.WithField("i", i).Debug("-> Publish message to Redis")
				defer self.Log.WithField("i", i).Debug("<- Publish message to Redis")
				return self.client.Publish(self.Ctx, self.channelName, payload).Err()
			})
		if err != nil {
			self.Log.WithError(err).Error("Persistant error to publish message, giving up")
			self.monitor.GetReport().RedisPublisher.Errors.PersistentFailure.Inc()
			return
		}

		self.monitor.GetReport().RedisPublisher.State.MessagesPublished.Inc()
		self.monitor.GetReport().RedisPublisher.State.LastSuccessfulMessageTimestamp.Store(time.Now().Unix())

		if self.redisConfig.MaxQueueSize > 3 {
			if self.GetWorkerQueueFillFactor() < 0.1 {
				self.Log.WithField("i", i).Debug("Redis queue almost empty")
			}
		}

		if self.redisConfig.MaxQueueSize > 3 {
			if self.GetWorkerQueueFillFactor() > 0.8 {
				self.Log.WithField("i", i).Warn("Redis queue is filling up")
			}
			if self.GetWorkerQueueFillFactor() > 0.99 {
				self.Log.WithField("i", i).Error("Redis queue is full")
			}
		}
		self.Log.WithField("i", i).Debug("...Redis publish done")
	}
	return nil
}
