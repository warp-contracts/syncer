package arweave

import (
	"context"
	"fmt"
	"strconv"
	"syncer/src/utils/build_info"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	client *resty.Client
}

func NewClient(url string) (self *Client, err error) {
	self = new(Client)
	self.client =
		resty.New().
			SetBaseURL(url).
			SetDebug(true).
			SetHeader("User-Agent", "warp.cc/syncer/"+build_info.Version).
			OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
				// Non-success status code turns into an error
				if resp.IsSuccess() {
					return nil
				}
				return fmt.Errorf("unexpected status: %s", resp.Status())
			})

	return
}

// https://docs.arweave.org/developers/server/http-api#network-info
func (self *Client) GetNetworkInfo(ctx context.Context) (out *NetworkInfo, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		SetResult(NetworkInfo{}).
		Get("info")
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

// Undocumented
func (self *Client) GetBlockByHeight(ctx context.Context, height int) (out *Block, err error) {
	resp, err := self.client.R().
		SetContext(ctx).
		SetResult(&Block{}).
		SetPathParam("height", strconv.Itoa(height)).
		Get("block/height/{height}")
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
		Get("tx/{id}")
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
