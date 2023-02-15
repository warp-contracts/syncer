package bundlr

import (
	"context"
	"fmt"
	"io"
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

func (self *Client) Upload(ctx context.Context, signer *Signer, item *BundleItem) (err error) {
	reader, err := item.Reader(signer)
	if err != nil {
		return
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	fmt.Println(body)
	// self.log.WithField("body", string(body)).Info("Item")

	resp, err := self.Request(ctx).
		SetBody(body).
		SetHeader("Content-Type", "application/octet-stream").
		// SetHeader("x-proof-type", "receipt").
		Post("/tx")
	if err != nil {
		return
	}

	self.log.WithField("resp", resp.Body()).Info("Uploaded to bundler")

	return
}
