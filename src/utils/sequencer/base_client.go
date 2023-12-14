package sequencer

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type BaseClient struct {
	config *config.Sequencer
	log    *logrus.Entry

	// State
	mtx     sync.RWMutex
	clients map[string]*resty.Client
	// limiters         map[string]*rate.Limiter
	currentClientIdx int
}

func newBaseClient(config *config.Sequencer) (self *BaseClient) {
	self = new(BaseClient)
	self.log = logger.NewSublogger("sequencer-client")
	self.config = config
	// self.limiters = make(map[string]*rate.Limiter)
	self.clients = make(map[string]*resty.Client)

	for _, url := range self.config.Urls {
		self.log.WithField("url", url).Debug("Creating client")
		self.clients[url] = resty.New().
			// SetDebug(true).
			SetBaseURL(url).
			SetTimeout(self.config.RequestTimeout).
			SetHeader("User-Agent", "warp.cc/sequencer").
			SetRetryCount(0).
			SetTransport(self.createTransport()).
			AddRetryCondition(self.onRetryCondition).
			// OnBeforeRequest(self.onRateLimit).
			// NOTE: Trace logs, used only when debugging. Needs to be before other OnAfterResponse callbacks
			// EnableTrace().
			// OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			// 	t, _ := json.Marshal(resp.Request.TraceInfo())
			// 	self.log.WithField("trace", string(t)).WithField("url", resp.Request.URL).Info("Trace")
			// 	return nil
			// }).
			// OnAfterResponse(self.onRetryRequest).
			OnAfterResponse(self.onStatusToError)
	}
	return
}

func (self *BaseClient) createTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   self.config.DialerTimeout,
		KeepAlive: self.config.DialerKeepAlive,
		DualStack: true,
	}

	return &http.Transport{
		// Some config options disable http2, try it anyway
		ForceAttemptHTTP2: true,

		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   self.config.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,

		// This is important. arweave.net may sometimes stop responding on idle connections,
		// resulting in error: context deadline exceeded (Client.Timeout exceeded while awaiting headers)
		IdleConnTimeout:     self.config.IdleConnTimeout,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     10,
	}
}

// Converts HTTP status to errors
func (self *BaseClient) onStatusToError(c *resty.Client, resp *resty.Response) error {
	// Non-success status code turns into an error
	if resp.IsSuccess() {
		return nil
	}
	if resp.StatusCode() > 399 && resp.StatusCode() < 500 {
		self.log.WithField("status", resp.StatusCode()).
			WithField("resp", string(resp.Body())).
			// WithField("body", resp.Request.Body).
			WithField("url", resp.Request.URL).
			Debug("Bad request")
	}
	return fmt.Errorf("unexpected status: %s", resp.Status())
}

// Retry request only upon server errors
func (self *BaseClient) onRetryCondition(resp *resty.Response, err error) bool {
	return resp != nil && resp.StatusCode() >= 500
}

// Handles rate limiting. There's one limiter per host
// func (self *BaseClient) onRateLimit(c *resty.Client, req *resty.Request) (err error) {
// 	var (
// 		limiter *rate.Limiter
// 		ok      bool
// 	)

// 	url, err := url.ParseRequestURI(req.URL)
// 	if err != nil {
// 		return
// 	}

// 	self.mtx.Lock()
// 	limiter, ok = self.limiters[url.Host]
// 	if !ok {
// 		limiter = rate.NewLimiter(rate.Every(self.config.LimiterInterval), self.config.LimiterBurstSize)
// 		self.limiters[url.Host] = limiter
// 	}
// 	self.mtx.Unlock()

// 	// Blocks till the request is possible
// 	// Or ctx gets canceled
// 	err = limiter.Wait(req.Context())
// 	if err != nil {
// 		d, _ := req.Context().Deadline()
// 		self.log.WithField("peer", url.Host).WithField("deadline", time.Until(d)).WithError(err).Error("Rate limiting failed")
// 	}
// 	return
// }

func (self *BaseClient) GetClient() *resty.Client {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.currentClientIdx = (self.currentClientIdx + 1) % len(self.clients)
	return self.clients[self.config.Urls[self.currentClientIdx]]
}
