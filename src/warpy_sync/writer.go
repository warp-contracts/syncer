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

	input           <-chan *[]InteractionPayload
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

func (self *Writer) WithInputChannel(v chan *[]InteractionPayload) *Writer {
	self.input = v
	return self
}

func (self *Writer) run() (err error) {
	for interactionPayload := range self.input {
		self.Log.Debug("Writer initialized")
		err = self.writeInteraction(interactionPayload)
	}
	return
}

func (self *Writer) writeInteraction(payloads *[]InteractionPayload) (err error) {
	chunkSize := self.Config.WarpySyncer.WriterInteractionChunkSize
	if len(*payloads) == 0 {
		self.Log.Debug("Interaction Payload slice is empty")
		return
	}
	members := make([]Member, 0, min(chunkSize, len(*payloads)))

	for i, payload := range *payloads {
		if payload.Points == 0 {
			self.Log.WithField("from_address", payload.FromAddress).Debug("Skipping from address, points 0")
			continue
		}

		roles, err := self.discordRoles(payload.FromAddress)
		if err != nil {
			self.Log.WithError(err).Error("Failed to get roles")
			return err
		}
		if roles == nil {
			self.Log.WithField("from_address", payload.FromAddress).Debug("Skipping address, not registered in warpy")
			continue
		}

		members = append(members, Member{Id: payload.FromAddress, Roles: *roles, Points: payload.Points})
		if len(members) >= chunkSize {
			err = self.sendInteractionChunk(&members)
			members = make([]Member, 0, min(chunkSize, len(*payloads)-i-1))
		}
		if err != nil {
			self.Log.WithError(err).Error("Failed to send interaction chunk")
			return err
		}
	}
	if len(members) > 0 {
		err = self.sendInteractionChunk(&members)
	}

	return
}

func (self *Writer) sendInteractionChunk(members *[]Member) (err error) {

	input := Input{
		Function: "addPointsForAddress",
		Points:   0,
		AdminId:  self.Config.WarpySyncer.SyncerInteractionAdminId,
		Members:  *members,
	}

	if len(*members) == 1 {
		self.Log.WithField("from_address", (*members)[0].Id).
			WithField("points", (*members)[0].Points).
			Debug("Writing interaction to Warpy...")

	} else {
		self.Log.WithField("chunk_size", len(*members)).
			WithField("points_default", 0).
			Debug("Writing interaction to Warpy...")
	}

	interactionId, err := warpy.WriteInteractionToWarpy(
		self.Ctx, self.Config.WarpySyncer.SyncerSigner, input, self.Config.WarpySyncer.SyncerContractId, self.Log, self.sequencerClient)
	if err != nil {
		return err
	}

	self.Log.WithField("interactionId", interactionId).Info("Interaction sent to Warpy")
	self.monitor.GetReport().WarpySyncer.State.WriterInteractionsToWarpy.Inc()
	return
}

func (self *Writer) discordRoles(fromAddress string) (roles *[]string, err error) {
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
			self.Log.WithError(err).WithField("from_address", fromAddress).
				Warn("Could not process assets sum, retrying...")
			return err
		}).
		Run(func() error {
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

			senderRoles, err := warpy.GetSenderRoles(self.httpClient, self.Config.WarpySyncer.SyncerWarpyApiUrl, senderDiscordId[0].Key, self.Log)

			if err != nil {
				self.Log.WithError(err).Warn("Could not retrieve sender roles")
				return err
			}

			if senderRoles != nil {
				roles = senderRoles
			} else {
				roles = &[]string{}
			}

			return nil
		})
	return
}
