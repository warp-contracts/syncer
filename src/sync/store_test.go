package sync

import (
	"context"

	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/monitor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	// "os"

	"testing"
)

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite
	ctx     context.Context
	cancel  context.CancelFunc
	config  *config.Config
	monitor *monitor.Monitor
}

func (s *StoreTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.ctx = common.SetConfig(s.ctx, s.config)
	s.monitor = monitor.NewMonitor()
}

func (s *StoreTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *StoreTestSuite) TestLifecycle() {
	store := NewStore(s.config, s.monitor)
	assert.NotNil(s.T(), store)

	err := store.Start()
	assert.Nil(s.T(), err)

	store.StopWait()

	<-store.Ctx.Done()
}
