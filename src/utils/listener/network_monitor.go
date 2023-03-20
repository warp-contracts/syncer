package listener

import (
	"context"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"time"
)

// Task that periodically checks for new arweave network info.
// Optionally waits for a number of required confirmation blocks before emitting the info
type NetworkMonitor struct {
	*task.Task

	// Runtime configuration
	requiredConfirmationBlocks int64
	client                     *arweave.Client
	monitor                    monitoring.Monitor

	// Output channel
	Output chan *arweave.NetworkInfo

	// Runtime variables
	lastHeight int64
}

// Using Arweave client periodically checks for blocks of transactions
func NewNetworkMonitor(config *config.Config) (self *NetworkMonitor) {
	self = new(NetworkMonitor)

	self.Output = make(chan *arweave.NetworkInfo)

	self.Task = task.NewTask(config, "network-monitor").
		WithOnAfterStop(func() {
			close(self.Output)
		})
	return
}

func (self *NetworkMonitor) WithMonitor(monitor monitoring.Monitor) *NetworkMonitor {
	self.monitor = monitor
	return self
}

func (self *NetworkMonitor) WithClient(client *arweave.Client) *NetworkMonitor {
	self.client = client
	return self
}

func (self *NetworkMonitor) WithRequiredConfirmationBlocks(requiredConfirmationBlocks int64) *NetworkMonitor {
	self.requiredConfirmationBlocks = requiredConfirmationBlocks
	return self
}

func (self *NetworkMonitor) WithInterval(interval time.Duration) *NetworkMonitor {
	self.Task = self.Task.WithPeriodicSubtaskFunc(interval, self.runPeriodically)
	return self
}

// Periodically checks Arweave network info for updated height
func (self *NetworkMonitor) runPeriodically() error {
	// Use a specific URL as the source of truth, to avoid race conditions with SDK
	ctx := context.WithValue(self.Ctx, arweave.ContextForcePeer, self.Config.ListenerNetworkInfoNodeUrl)
	ctx = context.WithValue(ctx, arweave.ContextDisablePeers, true)

	networkInfo, err := self.client.GetNetworkInfo(ctx)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get Arweave network info")
		self.monitor.GetReport().NetworkInfo.Errors.NetworkInfoDownloadErrors.Inc()
		return nil
	}

	self.monitor.GetReport().NetworkInfo.State.ArweaveCurrentHeight.Store(uint64(networkInfo.Height))
	self.monitor.GetReport().NetworkInfo.State.ArweaveLastNetworkInfoTimestamp.Store(uint64(time.Now().Unix()))

	// This is the last block height we consider stable
	stableHeight := networkInfo.Height - self.requiredConfirmationBlocks

	if stableHeight <= self.lastHeight {
		// Nothing changed, retry later
		return nil
	}

	// There are new blocks, broadcast
	self.lastHeight = stableHeight

	select {
	case <-self.StopChannel:
	case self.Output <- networkInfo:
	}

	return nil
}
