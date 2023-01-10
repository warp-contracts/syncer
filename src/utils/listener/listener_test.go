package listener

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

func TestListenerTestSuite(t *testing.T) {
	suite.Run(t, new(ListenerTestSuite))
}

type ListenerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
	log    *logrus.Entry
}

func (s *ListenerTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.log = logger.NewSublogger("listener-test")
}

func (s *ListenerTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ListenerTestSuite) TestLifecycle() {
	listener := NewListener(s.config).WithStartHeight(1082024)
	require.NotNil(s.T(), listener)

	listener.Start()
	time.Sleep(time.Minute)
	listener.StopWait()
}
