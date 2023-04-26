package arweave

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"syncer/src/utils/config"
	"time"

	"github.com/teivah/onecontext"
)

type Client struct {
	*BaseClient

	validateTag func(*Tag) error
}

func NewClient(ctx context.Context, config *config.Config) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(ctx, config)
	return
}

func (self *Client) WithTagValidator(v func(*Tag) error) *Client {
	self.validateTag = v
	return self
}

// https://docs.arweave.org/developers/server/http-api#network-info
func (self *Client) GetNetworkInfo(ctx context.Context) (out *NetworkInfo, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&NetworkInfo{}).
		ForceContentType("application/json").
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
		ForceContentType("application/json").
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
	ctx, cancel := context.WithTimeout(ctx, self.config.Arweave.CheckPeerTimeout)
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
		ForceContentType("application/json").
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
		SetError(&Error{}).
		ForceContentType("application/json").
		SetPathParam("id", id).
		Get("/tx/{id}")
	if resp.IsError() {
		msg, ok := resp.Error().(*Error)
		if !ok {
			err = ErrBadResponse
			return
		}

		if msg.Error != "" {
			err = errors.New(msg.Error)
			return
		}

		err = errors.New(string(http.StatusText(resp.StatusCode())))
		return
	}

	if err != nil {
		return
	}

	out, ok := resp.Result().(*Transaction)
	if !ok {
		if string(resp.Body()) == "Pending" {
			err = ErrPending
		} else {
			err = ErrFailedToParse
		}

		return
	}

	// Validate tags
	if self.validateTag != nil {
		for _, tag := range out.Tags {
			err = self.validateTag(&tag)
			if err != nil {
				return
			}
		}
	}

	return
}

// https://docs.arweave.org/developers/server/http-api#get-transaction-offset-and-size
func (self *Client) GetTransactionOffsetInfo(ctx context.Context, id string) (out *OffsetInfo, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&OffsetInfo{}).
		ForceContentType("application/json").
		SetPathParam("id", id).
		Get("/tx/{id}/offset")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*OffsetInfo)
	if !ok {
		err = ErrFailedToParse
		return
	}

	if !out.Offset.Valid || !out.Size.Valid {
		err = ErrBadResponse
		return
	}

	return
}

func (self *Client) getChunk(ctx context.Context, offset big.Int) (out *ChunkData, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetPathParam("offset", offset.String()).
		SetResult(&ChunkData{}).
		ForceContentType("application/json").
		Get("/chunk/{offset}")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*ChunkData)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}

func (self *Client) GetChunks(ctx context.Context, id string) (out bytes.Buffer, err error) {
	// Download chunks
	info, err := self.GetTransactionOffsetInfo(ctx, id)
	if err != nil {
		return
	}

	zero := big.NewInt(0)
	offset := &info.Offset.Int
	size := &info.Size.Int

	self.log.WithField("size", size.String()).Trace("Downloading")

	for size.Cmp(zero) > 0 {
		var chunk *ChunkData
		chunk, err = self.getChunk(ctx, *offset)
		if err != nil {
			return
		}

		out.Write(chunk.Chunk.Bytes())

		// Are there more chunks?
		chunkSize := big.NewInt(int64(len(chunk.Chunk.Bytes())))
		size = size.Sub(size, chunkSize)
		offset = offset.Sub(offset, chunkSize)
	}

	return
}

// https://docs.arweave.org/developers/server/http-api#get-transaction-field
func (self *Client) GetTransactionDataById(ctx context.Context, id string) (out bytes.Buffer, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetPathParam("id", id).
		Get("/tx/{id}/data")
	if err != nil {
		return
	}

	out.Write(resp.Body())
	if out.Len() > 0 {
		return
	}

	return self.GetChunks(ctx, id)
}
