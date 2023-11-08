package relay

import (
	"context"
	"errors"
	"sync"

	"github.com/cenkalti/backoff"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

// Produces a stream of Sequencer's blocks
// Blocks are put on the Output channel in order of height
// It uses Streamer to get new blocks from the Sequencer
// It uses Sequencer's REST API to download historical or missing blocks
// Handles gaps in the input stream
type Source struct {
	*task.Task

	input            <-chan *types.Block
	Output           chan *types.Block
	db               *gorm.DB
	monitor          monitoring.Monitor
	client           *rpchttp.HTTP
	lastSyncedHeight uint64
}

func NewSource(config *config.Config) (self *Source) {
	self = new(Source)

	self.Output = make(chan *types.Block, 1)

	self.Task = task.NewTask(config, "source").
		WithWorkerPool(config.Relayer.SourceMaxWorkers, config.Relayer.SourceMaxQueueSize).
		WithSubtaskFunc(self.run)

	return
}

func (self *Source) WithMonitor(monitor monitoring.Monitor) *Source {
	self.monitor = monitor
	return self
}

func (self *Source) WithDB(db *gorm.DB) *Source {
	self.db = db
	return self
}

func (self *Source) WithInputChannel(input <-chan *types.Block) *Source {
	self.input = input
	return self
}

func (self *Source) WithClient(client *rpchttp.HTTP) *Source {
	self.client = client
	return self
}

func (self *Source) initLastSyncedHeight() (err error) {
	var state model.State

	err = self.db.WithContext(self.Ctx).
		First(&state, model.SyncedComponentRelayer).
		Error
	if err == nil {
		// NO ERROR
		self.lastSyncedHeight = state.FinishedBlockHeight
		self.Log.WithField("height", self.lastSyncedHeight).Info("Found sync state of the relayer")
		return
	}

	// ERROR
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		self.Log.WithError(err).Error("Failed to get sync state of the relayer")
		return
	}

	// No record found, create one
	if !self.Config.IsDevelopment {
		// In production
		self.lastSyncedHeight = 0
	} else {
		// In development start syncing from the present block
		status, err := self.client.Status(self.Ctx)
		if err != nil {
			self.Log.Error("Failed to get status of the blockchain")
			return err
		}

		self.lastSyncedHeight = uint64(status.SyncInfo.LatestBlockHeight)
	}

	// Save the initial state to the database
	state = model.State{
		Name:                model.SyncedComponentRelayer,
		FinishedBlockHeight: self.lastSyncedHeight,
	}
	err = self.db.WithContext(self.Ctx).
		Table(model.TableState).
		Save(&state).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to update sync state after last block")
		return err
	}

	self.Log.WithField("height", self.lastSyncedHeight).Info("Initialized sync state of the relayer")

	return
}

func (self *Source) send(block *types.Block) (err error) {
	if self.lastSyncedHeight+1 != uint64(block.Height) {
		self.monitor.GetReport().Relayer.Errors.SequencerPermanentBlockDownloadError.Inc()
		err = errors.New("gap in the blocks stream")
		self.Log.WithField("last_synced_height", self.lastSyncedHeight).
			WithField("block_height", block.Height).
			WithError(err).
			Error("Gap in the blocks stream")
		return
	}

	select {
	case <-self.Ctx.Done():
		err = errors.New("task closing")
		return
	case self.Output <- block:
		self.lastSyncedHeight = uint64(block.Height)
	}

	self.monitor.GetReport().Relayer.State.SequencerBlocksDownloaded.Inc()

	return
}

// Downloads blocks in parallel
func (self *Source) download(len int) (err error) {
	out := make([]*types.Block, len)

	// Sync between workers
	var mtx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len)

	for i := 0; i < len; i++ {
		i := i
		height := int64(self.lastSyncedHeight) + int64(i+1)
		self.SubmitToWorker(func() {
			// Current height to download should
			var (
				errWorker error
				block     *ctypes.ResultBlock
			)
			// Retries downloading transaction until success or permanent error
			errWorker = task.NewRetry().
				WithContext(self.Ctx).
				WithMaxElapsedTime(self.Config.Relayer.SourceBackoffMaxElapsedTime).
				WithMaxInterval(self.Config.Relayer.SourceBackoffMaxInterval).
				WithAcceptableDuration(self.Config.Relayer.SourceBackoffMaxInterval * 10).
				WithOnError(func(err error, isDurationAcceptable bool) error {
					self.Log.WithError(err).WithField("height", height).Warn("Failed to download Sequencer's block, retrying after timeout")

					if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
						// Stopping
						return backoff.Permanent(err)
					}
					self.monitor.GetReport().Relayer.Errors.SequencerBlockDownloadError.Inc()

					return err
				}).
				Run(func() error {
					block, err = self.client.Block(self.Ctx, &height)
					return err
				})

			if err != nil {
				// Permanent error
				self.monitor.GetReport().Relayer.Errors.SequencerPermanentBlockDownloadError.Inc()
				self.Log.WithError(err).WithField("height", height).Error("Failed to download sequencer block, giving up")

				if !self.IsStopping.Load() {
					mtx.Lock()
					err = errWorker
					mtx.Unlock()
				}

				goto end
			}

			// Update monitoring
			self.monitor.GetReport().Relayer.State.SequencerBlocksCatchedUp.Inc()

			// Add to output
			mtx.Lock()
			out[i] = block.Block
			mtx.Unlock()

		end:
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	if err != nil {
		// There was a permanent error
		// This is serious, we cannot continue
		self.Log.WithError(err).Error("Permanent error in one of the workers, can't continue")
		return
	}

	// Put blocks into the Output channel
	for _, block := range out {
		err = self.send(block)
		if err != nil {
			return
		}
	}
	return
}

// Download blocks from last synced blocked to the specified height (inclusive; height is downloaded)
func (self *Source) catchUp(height int64) (err error) {
	len := height - int64(self.lastSyncedHeight)
	if len <= 0 {
		self.Log.WithField("last_synced_height", self.lastSyncedHeight).
			WithField("desired_height", height).
			Info("Catch up not possible")
		return
	}

	self.Log.WithField("last_synced_height", self.lastSyncedHeight).
		WithField("desired_height", height).
		WithField("num_blocks_to_catch_up", len).
		Info("Catching up")

	// Divide remaining blocks into batches that are downloaded in parallel and put into the Output channel
	// This is to avoid big pauses in block processing needed for downloading all blocks at once
	numBatches := int(len) / self.Config.Relayer.SourceBatchSize
	for i := 0; i < numBatches; i++ {
		err = self.download(self.Config.Relayer.SourceBatchSize)
		if err != nil {
			return
		}
	}

	// Last batch that contains less than SourceBatchSize blocks
	lastBatchSize := int(len) % self.Config.Relayer.SourceBatchSize
	if lastBatchSize != 0 {
		err = self.download(lastBatchSize)
		if err != nil {
			return
		}
	}

	return
}

func (self *Source) run() (err error) {
	err = self.initLastSyncedHeight()
	if err != nil {
		return
	}

	for block := range self.input {
		if uint64(block.Height) > self.lastSyncedHeight+1 {
			err = self.catchUp(block.Height - 1)
			if err != nil {
				if self.IsStopping.Load() {
					return nil
				}

				self.Log.WithError(err).
					WithField("last_synced_height", self.lastSyncedHeight).
					WithField("height", block.Height).
					Error("Failed to catch up")

				// This is a serious error, better stop the application
				panic(err)
			}
		}

		err = self.send(block)
		if err != nil {
			return
		}
	}

	return
}
