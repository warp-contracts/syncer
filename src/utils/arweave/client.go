package arweave

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syncer/src/utils/build_info"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type Client struct {
	client *resty.Client
	config *config.Config
	log    *logrus.Entry

	// State
	mtx   sync.RWMutex
	peers []string
}

func NewClient(config *config.Config) (self *Client) {
	self = new(Client)
	self.config = config
	self.log = logger.NewSublogger("arweave-client")

	// NOTE: Do not use SetBaseURL - it will break picking alternative peers upon error
	self.client =
		resty.New().
			// SetDebug(true).
			SetTimeout(self.config.ArRequestTimeout).
			SetHeader("User-Agent", "warp.cc/syncer/"+build_info.Version).
			SetRetryCount(2).
			SetLogger(NewLogger()).
			AddRetryAfterErrorCondition().
			OnAfterResponse(self.retryRequest).
			OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
				// Non-success status code turns into an error
				if resp.IsSuccess() {
					return nil
				}
				return fmt.Errorf("unexpected status: %s", resp.Status())
			})

	return
}

// Called upon error in the "main" client. Retries
func (self *Client) retryRequest(c *resty.Client, resp *resty.Response) (err error) {
	if resp.IsSuccess() {
		return nil
	}

	// Check if retrying is disabled through context
	isDisabled, ok := resp.Request.Context().Value(ContextDisablePeers).(bool)
	if ok && isDisabled {
		return nil
	}

	// Retry request for each of the alternate peerRetrying request with different peers.
	// Peers should already be ordered from the most usefull
	var (
		idx            int
		peer           string
		secondResponse *resty.Response
	)

	endpoint := strings.TrimPrefix(resp.Request.URL, self.config.ArNodeUrl)

	for {
		self.mtx.Lock()
		if idx >= len(self.peers) {
			// No more peers, report the first failure
			self.log.WithField("idx", idx).Info("No more peers to check")
			self.mtx.Unlock()
			return
		}
		peer = self.peers[idx]
		self.mtx.Unlock()

		// self.log.WithField("peer", peer).WithField("idx", idx).WithField("endpoint", endpoint).Info("Retrying request with different peer")

		// INFO[2023-01-11T22:30:06+01:00] Retrying request with different peer          endpoint="http://34.123.162.40:1984/tx/OKfCs5KVF_-vXQPH1isECWdXNlMgit494mnE6xpSdK8" module=warp.arweave-client peer="http://34.123.162.40:1984"

		// printnij te oba, peer się nie podmienia pewnie przez ten break w

		//	Make the same request, but change the URL
		resp.Request.URL = peer + endpoint
		secondResponse, err = resp.Request.Send()
		if err == nil {
			// Success
			break
		}

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

func (self *Client) url(endpoint string) (out string) {
	if endpoint[0:1] != "/" {
		endpoint = "/" + endpoint
	}
	return self.config.ArNodeUrl + endpoint
}

// Set the list of potential peers in order they should be used
// Uses only peers that are proper urls
func (self *Client) SetPeers(peers []string) {
	filtered := make([]string, 0, len(peers))

	for _, peer := range peers {
		_, err := url.Parse(peer)
		if err == nil {
			peer = strings.TrimSuffix(peer, "/")
			filtered = append(filtered, peer)
		}
	}

	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.peers = filtered
}

// https://docs.arweave.org/developers/server/http-api#network-info
func (self *Client) GetNetworkInfo(ctx context.Context) (out *NetworkInfo, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		// ForceContentType("application/json").
		SetResult(&NetworkInfo{}).
		Get(self.url("info"))
	if err != nil {
		return
	}

	out, ok := resp.Result().(*NetworkInfo)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}

// https://docs.arweave.org/developers/server/http-api#peer-list
func (self *Client) GetPeerList(ctx context.Context) (out []string, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult([]string{}).
		Get(self.url("peers"))
	if err != nil {
		return
	}

	peers, ok := resp.Result().(*[]string)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return *peers, nil
}

func (self *Client) CheckPeerConnection(ctx context.Context, peer string) (out *NetworkInfo, duration time.Duration, err error) {
	// Disable retrying request with different peer
	ctx = context.WithValue(ctx, ContextDisablePeers, true)

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
	defer cancel()

	resp, err := self.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&NetworkInfo{}).
		Get(peer + "/info")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*NetworkInfo)
	if !ok {
		err = ErrFailedToParse
		return
	}

	duration = resp.Time()

	return
}

// https://docs.arweave.org/developers/server/http-api#get-block-by-height
func (self *Client) GetBlockByHeight(ctx context.Context, height int64) (out *Block, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&Block{}).
		SetPathParam("height", strconv.FormatInt(height, 10)).
		Get(self.url("block/height/{height}"))
	if err != nil {
		return
	}

	out, ok := resp.Result().(*Block)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}

// https://docs.arweave.org/developers/server/http-api#get-transaction-by-id
func (self *Client) GetTransactionById(ctx context.Context, id string) (out *Transaction, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&Transaction{}).
		SetPathParam("id", id).
		Get(self.url("tx/{id}"))
	if err != nil {
		return
	}

	out, ok := resp.Result().(*Transaction)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}
