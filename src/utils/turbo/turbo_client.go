package turbo

import (
	"context"
	"io"
	"net/http"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/turbo/responses"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	*BaseClient
}

func NewClient(ctx context.Context, config *config.Bundlr) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(ctx, config)
	return
}

func (self *Client) Upload(ctx context.Context, item *bundlr.BundleItem) (out *responses.Upload, resp *resty.Response, err error) {
	reader, err := item.Reader()
	if err != nil {
		return
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err = req.
		SetBody(body).
		SetResult(&responses.Upload{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/octet-stream").
		Post("/tx")

	if resp != nil {
		if resp.StatusCode() == http.StatusAccepted {
			err = bundlr.ErrAlreadyReceived
			return
		}
		if resp.StatusCode() == http.StatusPaymentRequired {
			err = bundlr.ErrPaymentRequired
			return
		}
	}

	if err != nil {
		return
	}

	out, ok := resp.Result().(*responses.Upload)
	if !ok {
		err = bundlr.ErrFailedToParse
		return
	}

	return
}
