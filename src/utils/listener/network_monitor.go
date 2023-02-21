package listener

import (
	"context"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/monitor"
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
	monitor                    *monitor.Monitor

	// Output channel
	Output chan *arweave.NetworkInfo

	// Runtime variables
	lastHeight int64
}

// Using Arweave client periodically checks for blocks of transactions
func NewNetworkMonitor(config *config.Config, interval time.Duration) (self *NetworkMonitor) {
	self = new(NetworkMonitor)

	self.Output = make(chan *arweave.NetworkInfo, 1)

	self.Task = task.NewTask(config, "network-monitor").
		WithPeriodicSubtaskFunc(interval, self.runPeriodically)

	return
}

func (self *NetworkMonitor) WithMonitor(monitor *monitor.Monitor) *NetworkMonitor {
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

// Periodically checks Arweave network info for updated height
func (self *NetworkMonitor) runPeriodically() error {
	// Use a specific URL as the source of truth, to avoid race conditions with SDK
	ctx := context.WithValue(self.Ctx, arweave.ContextForcePeer, self.Config.ListenerNetworkInfoNodeUrl)
	ctx = context.WithValue(ctx, arweave.ContextDisablePeers, true)

	networkInfo, err := self.client.GetNetworkInfo(ctx)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get Arweave network info")
		if self.monitor != nil {
			self.monitor.Increment(monitor.Kind(monitor.NetworkInfoDownloadErrors))
		}
		return nil
	}

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
