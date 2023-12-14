package sequencer

import (
	"context"
	"io"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/sequencer/requests"
	"github.com/warp-contracts/syncer/src/utils/sequencer/responses"
	"github.com/warp-contracts/syncer/src/utils/sequencer/types"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	*BaseClient
}

func NewClient(config *config.Sequencer) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(config)
	return
}

func (self *Client) GetNonce(ctx context.Context, signatureType bundlr.SignatureType, owner string) (out *responses.GetNonce, resp *resty.Response, err error) {
	resp, err = self.GetClient().
		R().
		SetContext(ctx).
		SetBody(requests.GetNonce{
			SignatureType: signatureType,
			Owner:         owner,
		}).
		SetResult(&responses.GetNonce{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/json").
		EnableTrace().
		Post("/api/v1/nonce")
	if err != nil {
		return
	}

	// fmt.Printf("The request for nonce took %d ms\n", resp.Request.TraceInfo().TotalTime.Milliseconds())
	out, ok := resp.Result().(*responses.GetNonce)
	if !ok {
		err = ErrFailedToParse
		return
	}
	return
}

func (self *Client) UploadReader(ctx context.Context, reader io.Reader, mode types.BroadcastMode) (out *responses.Upload, resp *resty.Response, err error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	resp, err = self.GetClient().
		R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&responses.Upload{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("X-Broadcast-Mode", string(mode)).
		EnableTrace().
		Post("/api/v1/dataitem")
	if err != nil {
		return
	}

	// fmt.Printf("The request with the data item took %d ms\n", resp.Request.TraceInfo().TotalTime.Milliseconds())
	out, ok := resp.Result().(*responses.Upload)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}

func (self *Client) Upload(ctx context.Context, item *bundlr.BundleItem, mode types.BroadcastMode) (out *responses.Upload, resp *resty.Response, err error) {
	reader, err := item.Reader()
	if err != nil {
		return
	}

	return self.UploadReader(ctx, reader, mode)
}
