package bundlr

import (
	"context"
	"io"
	"syncer/src/utils/bundlr/responses"
	"syncer/src/utils/config"
)

type Client struct {
	*BaseClient
}

func NewClient(ctx context.Context, config *config.Bundlr) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(ctx, config)
	return
}

func (self *Client) Upload(ctx context.Context, signer *Signer, item *BundleItem) (out *responses.Upload, err error) {
	reader, err := item.Reader(signer)
	if err != nil {
		return
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	// fmt.Println(body)
	// self.log.WithField("body", string(body)).Info("Item")

	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetBody(body).
		SetResult(&responses.Upload{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/octet-stream").
		// SetHeader("x-proof-type", "receipt").
		Post("/tx")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*responses.Upload)
	if !ok {
		err = ErrFailedToParse
		return
	}

	// self.log.WithField("resp", resp.Body()).Info("Uploaded to bundler")

	return
}

func (self *Client) GetStatus(ctx context.Context, id string) (out *responses.Status, err error) {
	req, cancel := self.Request(ctx)
	defer cancel()

	resp, err := req.
		SetResult(&responses.Status{}).
		ForceContentType("application/json").
		SetPathParam("tx_id", id).
		Post("/tx/{tx_id}/status")
	if err != nil {
		return
	}

	out, ok := resp.Result().(*responses.Status)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}
