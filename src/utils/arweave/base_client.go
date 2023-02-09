package arweave

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"syncer/src/utils/build_info"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/teivah/onecontext"
	"golang.org/x/time/rate"
)

type BaseClient struct {
	client *resty.Client
	config *config.Config
	log    *logrus.Entry

	parentCtx context.Context

	// State
	mtx               sync.RWMutex
	peers             []string
	limiters          map[string]*rate.Limiter
	lastLimitDecrease time.Time

	ctx       context.Context
	cancel    context.CancelFunc
	lastReset time.Time
}

func newBaseClient(ctx context.Context, config *config.Config) (self *BaseClient) {
	self = new(BaseClient)
	self.config = config
	self.log = logger.NewSublogger("arweave-client")
	self.parentCtx = ctx

	self.limiters = make(map[string]*rate.Limiter)

	// Sets up HTTP client
	self.Reset()

	return
}

func (self *BaseClient) Reset() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	// Ensure client isn't reset too often
	if time.Since(self.lastReset) < self.config.ArLimiterDecreaseInterval {
		// Limit was just decreased
		return
	}
	self.lastReset = time.Now()

	if self.client != nil {
		self.log.Warn("Resetting HTTP client")
	}

	if self.cancel != nil {
		self.cancel()
	}

	self.ctx, self.cancel = context.WithCancel(self.parentCtx)

	// NOTE: Do not use SetBaseURL - it will break picking alternative peers upon error
	self.client =
		resty.New().
			SetTimeout(self.config.ArRequestTimeout).
			SetHeader("User-Agent", "warp.cc/syncer/"+build_info.Version).
			SetRetryCount(1).
			SetLogger(NewLogger(true /*force all logs to trace*/)).
			SetTransport(self.createTransport()).
			AddRetryCondition(self.onRetryCondition).
			// AddRetryAfterErrorCondition().
			OnBeforeRequest(self.onForcePeer).
			OnBeforeRequest(self.onRateLimit).
			// NOTE: Trace logs, used only when debugging. Needs to be before other OnAfterResponse callbacks
			// EnableTrace().
			// OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			// 	t, _ := json.Marshal(resp.Request.TraceInfo())
			// 	self.log.WithField("trace", string(t)).WithField("url", resp.Request.URL).Info("Trace")
			// 	return nil
			// }).
			OnAfterResponse(self.onRetryRequest).
			OnAfterResponse(self.onStatusToError)

	if config.Default().ArLimiterDecreaseFactor < 1 {
		self.client = self.client.OnAfterResponse(self.onTooManyRequests)
	}
}

func (self *BaseClient) createTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   self.config.ArDialerTimeout,
		KeepAlive: self.config.ArDialerKeepAlive,
		DualStack: true,
	}

	return &http.Transport{
		// Some config options disable http2, try it anyway
		ForceAttemptHTTP2: true,

		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   self.config.ArTLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,

		// This is important. arweave.net may sometimes stop responding on idle connections,
		// resulting in error: context deadline exceeded (Client.Timeout exceeded while awaiting headers)
		IdleConnTimeout:     self.config.ArIdleConnTimeout,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 1,
		MaxConnsPerHost:     1,
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
			WithField("body", resp.Request.Body).
			WithField("url", resp.Request.URL).
			Debug("Bad request")
	}
	return fmt.Errorf("unexpected status: %s", resp.Status())
}

// Handles HTTP 429 Too Many Requests - decreases limit for this hosts
func (self *BaseClient) onTooManyRequests(c *resty.Client, resp *resty.Response) error {
	if resp == nil || resp.StatusCode() != http.StatusTooManyRequests {
		// This isn't the case
		return nil
	}

	// Remote host receives too much requests, adjust rate limit
	url, err := url.ParseRequestURI(resp.Request.URL)
	if err != nil {
		self.log.WithError(err).Error("Failed to parse url")
		return err
	}

	self.mtx.Lock()
	defer self.mtx.Unlock()

	if time.Since(self.lastLimitDecrease) < self.config.ArLimiterDecreaseInterval {
		// Limit was just decreased
		return nil
	}

	self.lastLimitDecrease = time.Now()

	limiter, ok := self.limiters[url.Host]
	if !ok {
		err = errors.New("limiter not initialized")
		self.log.WithError(err).Error("Failed to parse url")
		return err
	}

	newLimit := limiter.Limit() * rate.Limit(self.config.ArLimiterDecreaseFactor)
	self.log.WithField("peer", url.Host).
		WithField("oldLimit", limiter.Limit()).
		WithField("newLimit", newLimit).
		Warn("Decreasing limit")
	limiter.SetLimit(newLimit)

	return nil
}

// Retry request only upon server errors
func (self *BaseClient) onRetryCondition(resp *resty.Response, err error) bool {
	return resp != nil && resp.StatusCode() >= 500
}

// Handles setting HOST url
// Properly handles cases when:
// - req.URL only contains the endpoint
// - req.URL contains the full URL
func (self *BaseClient) onForcePeer(c *resty.Client, req *resty.Request) (err error) {
	reqUrl, err := url.Parse(req.URL)
	if err != nil {
		return
	}

	peer, ok := req.Context().Value(ContextForcePeer).(string)
	if !ok {
		peer = self.config.ArNodeUrl
	}

	forcedUrl, err := url.Parse(peer)
	if err != nil {
		return
	}

	reqUrl.Host = forcedUrl.Host
	reqUrl.Scheme = forcedUrl.Scheme
	reqUrl.Path = forcedUrl.Path + reqUrl.Path

	// Path contains placeholder arguments in {} brackets
	// Using reqUrl.String() breaks the encoding
	req.URL = fmt.Sprintf("%s://%s%s", reqUrl.Scheme, reqUrl.Host, reqUrl.Path)

	return nil
}

// Handles rate limiting. There's one limiter per peer/hostname
func (self *BaseClient) onRateLimit(c *resty.Client, req *resty.Request) (err error) {
	self.log.Trace("Start rate limiter")
	defer self.log.Trace("Finish rate limiter")
	// Get the limiter, create it if needed
	var (
		limiter *rate.Limiter
		ok      bool
	)

	url, err := url.ParseRequestURI(req.URL)
	if err != nil {
		return
	}

	self.mtx.Lock()
	limiter, ok = self.limiters[url.Host]
	if !ok {
		limiter = rate.NewLimiter(rate.Every(self.config.ArLimiterInterval), self.config.ArLimiterBurstSize)
		self.limiters[url.Host] = limiter
	}
	self.mtx.Unlock()

	// Blocks till the request is possible
	// Or ctx gets canceled
	err = limiter.Wait(req.Context())
	if err != nil {
		d, _ := req.Context().Deadline()
		self.log.WithField("peer", url.Host).WithField("deadline", time.Until(d)).WithError(err).Error("Rate limiting failed")
	}
	return
}

// Handles retrying requests with alternative peers
func (self *BaseClient) onRetryRequest(c *resty.Client, resp *resty.Response) (err error) {
	if resp.IsSuccess() {
		return nil
	}

	// Check if retrying is disabled through context
	isDisabled, ok := resp.Request.Context().Value(ContextDisablePeers).(bool)
	if ok && isDisabled {
		return nil
	}

	// Disable retrying request with different peer
	ctx := context.WithValue(resp.Request.Context(), ContextDisablePeers, true)
	resp.Request.SetContext(ctx)

	// Retry request for each of the alternate peerRetrying request with different peers.
	// Peers should already be ordered from the most usefull
	var (
		idx            int
		peer           string
		secondResponse *resty.Response
	)

	endpoint := strings.Clone(strings.TrimPrefix(resp.Request.URL, self.config.ArNodeUrl))

	self.log.WithField("peer", peer).WithField("idx", idx).WithField("endpoint", endpoint).Info("Retrying begin")

	for {
		self.mtx.RLock()
		if idx >= len(self.peers) {
			// No more peers, report the first failure
			self.log.WithField("idx", idx).Info("No more peers to check")
			self.mtx.RUnlock()
			return
		}
		peer = self.peers[idx]
		self.mtx.RUnlock()

		self.log.WithField("peer", peer).WithField("idx", idx).WithField("endpoint", endpoint).Info("Retrying request with different peer")

		//	Make the same request, but change the URL
		resp.Request.URL = peer + endpoint
		secondResponse, err = resp.Request.Send()
		if err == nil {
			// Success
			break
		}

		self.log.WithError(err).Debug("Retried request failed")

		// Handle timeout in context
		if secondResponse.Request.Context().Err() != nil {
			err = secondResponse.Request.Context().Err()
			return
		}
		idx += 1
	}
	// Replace the response returned to the API user
	(*resp) = (*secondResponse)
	return err
}

// Set the list of potential peers in order they should be used
// Uses only peers that are proper urls
func (self *BaseClient) SetPeers(peers []string) {
	filtered := make([]string, 0, len(peers))

	for _, peer := range peers {
		_, err := url.Parse(peer)
		if err == nil {
			peer = strings.TrimSuffix(peer, "/")
			filtered = append(filtered, peer)
		}
	}

	// Cleanup idle connections
	self.client.GetClient().CloseIdleConnections()

	self.mtx.Lock()
	self.peers = filtered
	self.mtx.Unlock()
}

func (self *BaseClient) Request(ctx context.Context) *resty.Request {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	ctx, _ = onecontext.Merge(self.ctx, ctx)

	return self.client.R().
		SetContext(ctx).
		ForceContentType("application/json")
}
