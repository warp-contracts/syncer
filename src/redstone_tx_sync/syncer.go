package redstone_tx_sync

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	sequencer_types "github.com/warp-contracts/syncer/src/utils/sequencer/types"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Syncer struct {
	*task.Task
	monitor         monitoring.Monitor
	input           chan *BlockInfoPayload
	Output          chan *LastSyncedBlockPayload
	sequencerClient *sequencer.Client
}

// This task receives block info in the input channel, iterate through all of the block's transactions in order to check if any of it contains
// Redstone data and if so - writes an interaction to Warpy. It emits block height and block hash in the Output channel
func NewSyncer(config *config.Config) (self *Syncer) {
	self = new(Syncer)

	self.Output = make(chan *LastSyncedBlockPayload)

	self.Task = task.NewTask(config, "syncer").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.RedstoneTxSyncer.SyncerNumWorkers, config.RedstoneTxSyncer.SyncerWorkerQueueSize)

	return
}

func (self *Syncer) WithInputChannel(v chan *BlockInfoPayload) *Syncer {
	self.input = v
	return self
}

func (self *Syncer) WithMonitor(monitor monitoring.Monitor) *Syncer {
	self.monitor = monitor
	return self
}

func (self *Syncer) WithSequencerClient(sequencerClient *sequencer.Client) *Syncer {
	self.sequencerClient = sequencerClient
	return self
}

func (self *Syncer) run() (err error) {
	for block := range self.input {
		self.Log.WithField("height", block.Height).Debug("Checking transactions for block")
		var wg sync.WaitGroup
		wg.Add(len(block.Transactions))

		for _, tx := range block.Transactions {
			tx := tx
			block := block
			self.SubmitToWorker(func() {
				err := self.checkTxAndWriteInteraction(tx, block)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
						Error("Could not process transaction")
					self.monitor.GetReport().RedstoneTxSyncer.Errors.SyncerWriteInteractionFailures.Inc()
					goto end
				}

				self.monitor.GetReport().RedstoneTxSyncer.State.SyncerTxsProcessed.Inc()

			end:
				wg.Done()
			})
		}

		wg.Wait()

		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- &LastSyncedBlockPayload{
			Height: block.Height,
			Hash:   block.Hash,
		}:
		}

		self.monitor.GetReport().RedstoneTxSyncer.State.SyncerBlocksProcessed.Inc()
	}
	return
}

func (self *Syncer) checkTxAndWriteInteraction(tx *types.Transaction, block *BlockInfoPayload) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.RedstoneTxSyncer.SyncerBackoffInterval).
		WithAcceptableDuration(self.Config.RedstoneTxSyncer.SyncerBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().RedstoneTxSyncer.Errors.SyncerWriteInteractionFailures.Inc()
			self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
				Warn("Could not process transaction, retrying...")
			return err
		}).
		Run(func() error {
			txContainsRedstoneData := self.checkTxForData(tx, self.Config.RedstoneTxSyncer.SyncerRedstoneData, self.Ctx)
			if txContainsRedstoneData {
				self.Log.WithField("txId", tx.Hash()).WithField("height", block.Height).Info("Found new Redstone tx")
				sender, err := self.getTxSenderHash(tx)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).Warn("Could not retrieve tx sender")
					return err
				}
				input := Input{
					Function: "addPointsCsv",
					Points:   self.Config.RedstoneTxSyncer.SyncerInteractionPoints,
					AdminId:  self.Config.RedstoneTxSyncer.SyncerInteractionAdminId,
					Members:  []Member{{Id: sender, Roles: []string{}}},
					NoBoost:  true,
				}
				self.Log.WithField("txId", tx.Hash()).Debug("Writing interaction to Warpy...")
				interactionId, err := self.writeInteractionToWarpy(
					self.Ctx, tx, self.Config.RedstoneTxSyncer.SyncerSigner, input, self.Config.RedstoneTxSyncer.SyncerContractId)
				if err != nil {
					return err
				}
				self.Log.WithField("interactionId", interactionId).Info("Interaction sent to Warpy")
				self.monitor.GetReport().RedstoneTxSyncer.State.SyncerInteractionsToWarpy.Inc()
			}

			return err
		})

	return
}

func (self *Syncer) checkTxForData(tx *types.Transaction, data string, ctx context.Context) (txContainsData bool) {
	txContainsData = false
	encodedString := hex.EncodeToString(tx.Data())
	if strings.Contains(encodedString, data) {
		txContainsData = true
	}
	return
}

func (self *Syncer) getTxSenderHash(tx *types.Transaction) (txSenderHash string, err error) {
	signer := types.LatestSignerForChainID(tx.ChainId())
	sender, err := types.Sender(signer, tx)
	txSenderHash = sender.Hash().String()
	return
}

func (self *Syncer) writeInteractionToWarpy(ctx context.Context, tx *types.Transaction, arweaveSigner string, input json.Marshaler, contractId string) (interactionId string, err error) {
	signer, err := bundlr.NewArweaveSigner(arweaveSigner)
	if err != nil {
		self.Log.WithError(err).Error("Could not create Arweave Signer")
		return
	}

	interactionId, err = self.sequencerClient.UploadInteraction(ctx, input, sequencer_types.WriteInteractionOptions{ContractTxId: contractId}, signer)
	if err != nil {
		self.Log.WithError(err).Error("Could not write interaction to Warpy")
		return
	}
	return
}
