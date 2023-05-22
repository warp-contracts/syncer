package bundlr

import (
	"context"

	"github.com/warp-contracts/syncer/src/utils/config"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	signer *Signer
	client *Client
}

func (s *ClientTestSuite) SetupSuite() {
	var err error
	s.signer, err = NewSigner(EMPTY_ARWEAVE_WALLET)
	require.Nil(s.T(), err)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.client = NewClient(s.ctx, &config.Default().Bundlr)
}

func (s *ClientTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ClientTestSuite) TestUpload() {
	item := &BundleItem{
		Data: []byte("asdf"),
		Tags: Tags{Tag{Name: "name", Value: "value"}},
	}
	resp, r, err := s.client.Upload(s.ctx, s.signer, item)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), r)
	require.NotNil(s.T(), resp)

	status, err := s.client.GetStatus(s.ctx, resp.Id)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), status)
}
