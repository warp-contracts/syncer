package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"sync"
	"sync/atomic"
	"syncer/src/utils/arweave"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"

	"time"

	"github.com/gammazero/workerpool"
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

	// State
	blacklisted    sync.Map
	numBlacklisted atomic.Int64

	// Worker pool for checking peers in parallel
	workers *workerpool.WorkerPool
}

type metric struct {
	Peer     string
	Height   int64
	Blocks   int64
	Duration time.Duration
}

func (self *metric) String() string {
	b, _ := json.Marshal(self)
	return string(b)
}

func NewPeerMonitor(config *config.Config) (self *PeerMonitor) {
	self = new(PeerMonitor)
	self.log = logger.NewSublogger("peer-monitor")
	self.config = config

	// PeerMonitor context, active as long as there's anything running in PeerMonitor
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Stopping
	self.stopOnce = &sync.Once{}
	self.isStopping = &atomic.Bool{}
	self.stopWaitGroup = sync.WaitGroup{}
	self.stopChannel = make(chan bool, 1)

	// Worker pool for checking peers in parallel
	self.workers = workerpool.New(50)
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
	var timer *time.Timer

	f := func() {
		// Setup waiting before the next check
		defer func() { timer = time.NewTimer(self.config.PeerMonitorPeriod) }()

		peers, err := self.getPeers()
		if err != nil {
			self.log.WithError(err).Error("Failed to get peers")
			return
		}

		peers = self.sortPeersByMetrics(peers)

		self.client.SetPeers(peers[:15])

		self.log.WithField("numBlacklisted", self.numBlacklisted.Load()).Info("Set new peers")

		self.cleanupBlacklist()

	}

	for {
		// Start monitoring peers right away
		f()
		select {
		case <-self.stopChannel:
			return nil
		case <-timer.C:
			f()
		}
	}
}

func (self *PeerMonitor) getPeers() (peers []string, err error) {
	// Get peers available in the network
	allPeers, err := self.client.GetPeerList(self.Ctx)
	if err != nil {
		self.log.Error("Failed to get Arweave network info")
		return
	}
	self.log.WithField("numPeers", len(allPeers)).Debug("Got peers")

	// Slice of checked addresses in proper format
	peers = make([]string, 0, len(allPeers))

	var addr netip.AddrPort
	for _, peer := range allPeers {
		// Validate peer address
		addr, err = netip.ParseAddrPort(peer)
		if err != nil || !addr.IsValid() {
			self.log.WithField("peer", peer).Error("Bad peer address")
			continue
		}

		// Peers are in format <ip>:<port>, add http schema
		peers = append(peers, "http://"+peer)
	}

	return
}

func (self *PeerMonitor) sortPeersByMetrics(allPeers []string) (peers []string) {
	self.log.Debug("Checking peers")

	// Sync between workers
	var mtx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(allPeers))

	// Metrics used to sort peers
	metrics := make([]*metric, 0, len(allPeers))

	// Perform test requests
	for i, peer := range allPeers {
		// Copy variables
		peer := peer
		i := i

		self.workers.Submit(func() {
			var (
				info     *arweave.NetworkInfo
				duration time.Duration
				err      error
			)

			// Neglect blacklisted peers
			if _, ok := self.blacklisted.Load(peer); ok {
				goto end
			}

			self.log.WithField("peer", peer).WithField("idx", i).WithField("maxIdx", len(allPeers)-1).Trace("Checking peer")
			info, duration, err = self.client.CheckPeerConnection(self.Ctx, peer)
			if err != nil {
				self.log.WithField("peer", peer).Trace("Black list peer")

				// Put the peer on the blacklist
				self.blacklisted.Store(peer, time.Now())
				self.numBlacklisted.Add(1)

				// Neglect peers that returned and error
				goto end
			}

			mtx.Lock()
			metrics = append(metrics, &metric{
				Duration: duration,
				Height:   info.Height,
				Blocks:   info.Blocks,
				Peer:     peer,
			})

			mtx.Unlock()

			// self.log.WithField("metric", metrics[len(metrics)-1]).Info("Metric")

		end:
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	// self.log.WithField("metrics", len(metrics)).WithField("metrics", metrics).Debug("Metrics")

	// Sort using response times (less is better)
	sort.Slice(metrics, func(i int, j int) bool {
		return metrics[i].Duration < metrics[j].Duration
	})

	// Sort using number of not synchronized blocks (less is better)
	sort.SliceStable(metrics, func(i int, j int) bool {
		a := metrics[i]
		b := metrics[j]
		return (a.Height-a.Blocks < b.Height-b.Blocks)
	})

	// Get the sorted peers
	peers = make([]string, len(metrics))
	for i, metric := range metrics {
		peers[i] = metric.Peer
	}

	return
}

func (self *PeerMonitor) cleanupBlacklist() {
	now := time.Now()
	numRemoved := 0

	self.blacklisted.Range(func(peer, timeWhenAdded any) bool {
		t, ok := timeWhenAdded.(time.Time)
		if !ok {
			self.log.Error("Bad value stored in the blacklist")
			return false
		}

		if now.Sub(t) >= self.config.PeerMonitorMaxTimeBlacklisted {
			// Peer has been blacklisted long enough
			self.blacklisted.Delete(peer)
			self.numBlacklisted.Add(-1)
			self.log.WithField("peer", peer).Debug("Removed peer from blacklist")
		}

		if numRemoved >= self.config.PeerMonitorMaxPeersRemovedFromBlacklist {
			// Maximum of blacklisted peers got removed, rest will be removed later
			return false
		}

		return true
	})
}

func (self *PeerMonitor) Stop() {
	self.log.Info("Stopping PeerMonitor...")
	self.stopOnce.Do(func() {
		// Signals that we're stopping
		close(self.stopChannel)

		// Mark that we're stopping
		self.isStopping.Store(true)

		self.workers.Stop()
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
