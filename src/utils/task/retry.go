package task

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Implement operation retrying
type Retry struct {
	maxElapsedTime time.Duration
	maxInterval    time.Duration
	onError        func(error)
}

func NewRetry() *Retry {
	return new(Retry)
}

func (self *Retry) WithMaxElapsedTime(maxElapsedTime time.Duration) *Retry {
	self.maxElapsedTime = maxElapsedTime
	return self
}

func (self *Retry) WithMaxInterval(maxInterval time.Duration) *Retry {
	self.maxInterval = maxInterval
	return self
}

func (self *Retry) WithOnError(v func(error)) *Retry {
	self.onError = v
	return self
}

func (self *Retry) onNotify(err error, duration time.Duration) {
	self.onError(err)
}

func (self *Retry) Run(f func() error) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = self.maxElapsedTime
	b.MaxInterval = self.maxInterval
	return backoff.RetryNotify(f, b, self.onNotify)
}
