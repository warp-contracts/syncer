package arweave

import (
	"context"
	"strconv"
	"syncer/src/utils/config"
	"time"

	"github.com/teivah/onecontext"
)

type Client struct {
	*BaseClient
}

func NewClient(ctx context.Context, config *config.Config) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(ctx, config)
	return
}

// https://docs.arweave.org/developers/server/http-api#network-info
func (self *Client) GetNetworkInfo(ctx context.Context) (out *NetworkInfo, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&NetworkInfo{}).
		Get("/info")
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
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult([]string{}).
		Get("/peers")
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
	ctx = context.WithValue(ctx, ContextForcePeer, peer)

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, self.config.ArCheckPeerTimeout)
	defer cancel()

	self.mtx.RLock()
	ctx, _ = onecontext.Merge(self.ctx, ctx)
	self.mtx.RUnlock()

	resp, err := self.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&NetworkInfo{}).
		Get("/info")
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
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&Block{}).
		SetPathParam("height", strconv.FormatInt(height, 10)).
		Get("/block/height/{height}")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*Block)
	if !ok {
		err = ErrFailedToParse
		return
	}

	// self.log.WithField("block", string(resp.Body())).Info("Block")

	return
}

// https://docs.arweave.org/developers/server/http-api#get-transaction-by-id
func (self *Client) GetTransactionById(ctx context.Context, id string) (out *Transaction, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&Transaction{}).
		SetPathParam("id", id).
		Get("/tx/{id}")
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
