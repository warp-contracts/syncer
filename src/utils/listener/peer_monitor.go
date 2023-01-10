package listener

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"syncer/src/utils/arweave"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/sirupsen/logrus"
)

type PeerMonitor struct {
	client *arweave.Client
	config *config.Config
	log    *logrus.Entry

	// Stopping
	isStopping    *atomic.Bool
	stopChannel   chan bool
	stopOnce      *sync.Once
	stopWaitGroup sync.WaitGroup
	Ctx           context.Context
	cancel        context.CancelFunc
}

type metric struct {
	BlockHeight int64
	Duration    time.Duration
}

func NewPeerMonitor(config *config.Config) (self *PeerMonitor) {
	self = new(PeerMonitor)
	self.log = logger.NewSublogger("listener")
	self.config = config

	// PeerMonitor context, active as long as there's anything running in PeerMonitor
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Stopping
	self.stopOnce = &sync.Once{}
	self.isStopping = &atomic.Bool{}
	self.stopWaitGroup = sync.WaitGroup{}
	self.stopChannel = make(chan bool, 1)

	return
}

func (self *PeerMonitor) WithClient(client *arweave.Client) *PeerMonitor {
	self.client = client
	return self
}

func (self *PeerMonitor) Start() {
	go func() {
		defer func() {
			// run() finished, so it's time to cancel Store's context
			// NOTE: This should be the only place self.Ctx is cancelled
			self.cancel()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in Store. Stopping.")
				panic(p)
			}
		}()
		err := self.run()
		if err != nil {
			self.log.WithError(err).Error("Error in run()")
		}
	}()
}

// Periodically checks Arweave network info for updated height
func (self *PeerMonitor) run() (err error) {
	ticker := time.NewTicker(self.config.StoreMaxTimeInQueue)

	for {
		select {
		case <-self.stopChannel:
			return nil
		case <-ticker.C:
			peers, err := self.client.GetPeerList(self.Ctx)
			if err != nil {
				self.log.Error("Failed to get Arweave network info")
				continue
			}
			self.log.WithField("numPeers", len(peers)).WithField("peers", peers).Debug("Got peers")

			for i, peer := range peers {
				peers[i] = "https://" + peer
			}
			self.updateMetrics(peers)
		}
	}
}

func (self *PeerMonitor) updateMetrics(peers []string) {
	// Get the metrics
	metrics := make([]*metric, len(peers))
	for i, peer := range peers {
		self.log.WithField("peer", peer).Debug("Checking peer")
		info, duration, err := self.client.CheckPeerConnection(self.Ctx, peer)
		if err != nil {
			self.log.WithField("peer", peer).Error("Failed to check peer")
			continue
		}
		metrics[i] = &metric{
			Duration:    duration,
			BlockHeight: info.Height,
		}
	}

	sort.Slice(peers, func(i int, j int) bool {
		return metrics[i].BlockHeight > metrics[j].BlockHeight || metrics[i].Duration < metrics[j].Duration
	})

	self.client.SetPeers(peers)
}

func (self *PeerMonitor) Stop() {
	self.log.Info("Stopping PeerMonitor...")
	self.stopOnce.Do(func() {
		// Signals that we're stopping
		close(self.stopChannel)

		// Mark that we're stopping
		self.isStopping.Store(true)
	})
}

func (self *PeerMonitor) StopWait() {
	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	self.Stop()

	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, failed to finish PeerMonitor")
	case <-self.Ctx.Done():
		self.log.Info("PeerMonitor finished")
	}
}
