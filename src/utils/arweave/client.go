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
			self.mtx.Unlock()
			return
		}
		peer = self.peers[idx]
		self.mtx.Unlock()

		self.log.WithField("peer", peer).Info("Retrying request with different peer")

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
		SetResult(NetworkInfo{}).
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
	resp, err := self.client.R().
		SetContext(ctx).
		SetResult(NetworkInfo{}).
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
