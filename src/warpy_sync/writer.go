package warpy_sync

import (
	"context"
	"errors"

	"github.com/cenkalti/backoff"
	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warpy"
)

type Writer struct {
	*task.Task

	monitor monitoring.Monitor

	input           <-chan *InteractionPayload
	sequencerClient *sequencer.Client
	httpClient      *resty.Client
}

func NewWriter(config *config.Config) (self *Writer) {
	self = new(Writer)

	self.httpClient = resty.New().
		SetTimeout(config.WarpySyncer.WriterHttpRequestTimeout)

	self.Task = task.NewTask(config, "writer").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.Evolver.DownloaderNumWorkers, config.Evolver.DownloaderWorkerQueueSize)

	return
}

func (self *Writer) WithMonitor(monitor monitoring.Monitor) *Writer {
	self.monitor = monitor
	return self
}

func (self *Writer) WithSequencerClient(sequencerClient *sequencer.Client) *Writer {
	self.sequencerClient = sequencerClient
	return self
}

func (self *Writer) WithInputChannel(v chan *InteractionPayload) *Writer {
	self.input = v
	return self
}

func (self *Writer) run() (err error) {
	for interactionPayload := range self.input {
		self.Log.WithField("from_address", interactionPayload.FromAddress).WithField("sum", interactionPayload.Points).Debug("Writing interaction initialized")
		err = self.writeInteraction(interactionPayload.FromAddress, interactionPayload.Points)
	}
	return
}

func (self *Writer) writeInteraction(fromAddress string, points int64) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.WarpySyncer.WriterBackoffInterval).
		WithAcceptableDuration(self.Config.WarpySyncer.WriterBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().WarpySyncer.Errors.WriterFailures.Inc()
			self.Log.WithError(err).WithField("from_address", fromAddress).WithField("points", points).
				Warn("Could not process assets sum, retrying...")
			return err
		}).
		Run(func() error {
			// TO BE REMOVED
			if fromAddress != "0x825999DB01C9D7b9A96411FfAd24a6Db6e11dC0c" {
				return nil
			}

			senderDiscordIdPayload, err := warpy.GetSenderDiscordId(self.httpClient, self.Config.WarpySyncer.SyncerDreUrl, fromAddress, self.Log)
			if err != nil {
				self.Log.WithError(err).Warn("Could not retrieve sender Discord id")
				return err
			}

			if senderDiscordIdPayload == nil || len(*senderDiscordIdPayload) == 0 {
				self.Log.WithField("from_address", fromAddress).
					Info("Address not registered in Warpy, exiting")
				return nil
			}

			senderDiscordId := []model.SenderDiscordIdPayload{}
			senderDiscordId = append(senderDiscordId, *senderDiscordIdPayload...)

			roles := []string{}
			senderRoles, err := warpy.GetSenderRoles(self.httpClient, self.Config.WarpySyncer.SyncerWarpyApiUrl, senderDiscordId[0].Key, self.Log)

			if err != nil {
				self.Log.WithError(err).Warn("Could not retrieve sender roles")
				return err
			}

			if senderRoles != nil {
				roles = append(roles, *senderRoles...)
			}

			input := Input{
				Function: "addPointsForAddress",
				Points:   points,
				AdminId:  self.Config.WarpySyncer.SyncerInteractionAdminId,
				Members:  []Member{{Id: fromAddress, Roles: roles}},
			}
			self.Log.WithField("from_address", fromAddress).Debug("Writing interaction initialized")
			interactionId, err := warpy.WriteInteractionToWarpy(
				self.Ctx, self.Config.WarpySyncer.SyncerSigner, input, self.Config.WarpySyncer.SyncerContractId, self.Log, self.sequencerClient)
			if err != nil {
				return err
			}
			self.Log.WithField("interactionId", interactionId).Info("Interaction sent to Warpy")
			self.monitor.GetReport().WarpySyncer.State.WriterInteractionsToWarpy.Inc()

			return err
		})

	return
}
