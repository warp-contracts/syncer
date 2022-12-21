package arweave

import (
	"context"

	"syncer/src/utils/config"
	"syncer/src/utils/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	// "os"

	"testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
	client *Client
	log    *logrus.Entry
}

func (s *ClientTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.log = logger.NewSublogger("arweave-test")

	var err error
	s.client, err = NewClient("https://arweave.net")
	require.NotNil(s.T(), s.client)
	require.Nil(s.T(), err)
}

func (s *ClientTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ClientTestSuite) TestGetNetworkInfo() {
	out, err := s.client.GetNetworkInfo(s.ctx)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), out)
	require.NotZero(s.T(), out.Blocks)
}

func (s *ClientTestSuite) TestGetBlockByHeight() {
	out, err := s.client.GetBlockByHeight(s.ctx, 1082024)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), out)
	require.NotZero(s.T(), out.Height)
	require.Greater(s.T(), len(out.Txs), 0)
}

func (s *ClientTestSuite) TestGetTransactionById() {
	out, err := s.client.GetTransactionById(s.ctx, "EOlTGnmAsif6zpZgzTGBP268bit_UIW3QX7uUx874PA")
	require.Nil(s.T(), err)
	require.NotNil(s.T(), out)
	require.NotZero(s.T(), out.ID)

}
