package poller

import (
	"context"
	"time"

	"syncer/src/utils/config"
	"syncer/src/utils/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	// "os"

	"testing"
)

func TestPollerTestSuite(t *testing.T) {
	suite.Run(t, new(PollerTestSuite))
}

type PollerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
	log    *logrus.Entry
}

func (s *PollerTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.log = logger.NewSublogger("poller-test")
}

func (s *PollerTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *PollerTestSuite) TestLifecycle() {
	poller := NewPoller(s.config).WithStartHeight(1082024)
	require.NotNil(s.T(), poller)

	poller.Start()
	time.Sleep(time.Minute)
	poller.StopWait()
}
