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

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

type ControllerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
}

func (s *ControllerTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.config = config.Default()
	s.ctx = common.SetConfig(s.ctx, s.config)
}

func (s *ControllerTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ControllerTestSuite) TestLifecycle() {
	controller, err := NewController(s.config)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), controller)
	// controller.Start()
	// time.Sleep(time.Second)
	// controller.StopSync()
}
