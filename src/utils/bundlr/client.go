package bundlr

import (
	"context"
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

func (self *Client) Upload(ctx context.Context, item *BundleItem) {
	_, err := self.Request(ctx).
		SetBody(item).
		Post("/tx")
	if err != nil {
		return
	}

}
