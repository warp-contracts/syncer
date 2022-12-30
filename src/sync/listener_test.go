package sync

import (
	"context"

	"syncer/src/utils/common"
	"syncer/src/utils/config"

	"github.com/stretchr/testify/assert"
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
}

func (s *ListenerTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.ctx = common.SetConfig(s.ctx, s.config)
}

func (s *ListenerTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ListenerTestSuite) TestLifecycle() {
	listener := NewListener(s.config)
	assert.NotNil(s.T(), listener)
	// listener.Start(10000)
	// time.Sleep(time.Second * 15)
	// listener.StopWait()
	// <-listener.Ctx.Done()
}
