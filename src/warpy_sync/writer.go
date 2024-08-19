package warpy_sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/config"
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

	payloadChunk := make([]InteractionPayload, 0, chunkSize)

	var cap int64
	for i, payload := range *payloads {
		if payload.Points == 0 {
			self.Log.WithField("from_address", payload.FromAddress).Debug("Skipping from address, points 0")
			continue
		}

		payloadChunk = append(payloadChunk, payload)
		if len(payloadChunk) >= chunkSize {
			if i == len(*payloads)-1 {
				cap = self.calculateCap()
			}
			err = self.sendInteractionChunk(&payloadChunk, cap)
			if err != nil {
				self.Log.
					WithField("chunk_size", len(payloadChunk)).
					WithError(err).Error("Failed to send interaction chunk")
				return err
			}
			payloadChunk = make([]InteractionPayload, 0, chunkSize)
		}

	}
	if len(payloadChunk) > 0 {
		cap := self.calculateCap()
		err = self.sendInteractionChunk(&payloadChunk, cap)
		if err != nil {
			self.Log.WithError(err).Error("Failed to send interaction chunk")
			return err
		}
	}

	return
}

func (self *Writer) sendInteractionChunk(interactions *[]InteractionPayload, cap int64) (err error) {
	self.Log.WithField("chunk_size", len(*interactions)).
		Info("Attempting to send a chunk of interaction payload")

	addressToRoles, err := self.walletAddressToDiscordRoles(interactions)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get roles")
		return err
	}
	if addressToRoles == nil || len(*addressToRoles) == 0 {
		self.Log.WithField("chunk_size", len(*interactions)).
			Debug("Skipping writing interactions, none of the addresses registered in warpy")
		return
	}

	members := make([]Member, 0, len(*interactions))
	for _, p := range *interactions {
		roles, ok := (*addressToRoles)[p.FromAddress]
		if roles == nil {
			roles = make([]string, 0)
		}
		if ok {
			members = append(members, Member{Id: p.FromAddress, Roles: roles, Points: p.Points})
		} else {
			self.Log.WithField("from_address", p.FromAddress).Debug("Skipping address, not registered in warpy")
		}
	}

	if len(members) == 1 {
		self.Log.WithField("from_address", (members)[0].Id).
			WithField("points", (members)[0].Points).
			Debug("Writing interaction to Warpy...")
	} else {
		self.Log.WithField("chunk_size", len(members)).
			WithField("points_default", 0).
			Debug("Writing interaction to Warpy...")
	}

	input := Input{
		Function: "addPointsWithCap",
		Points:   0,
		AdminId:  self.Config.WarpySyncer.SyncerInteractionAdminId,
		Members:  members,
		Cap:      cap,
	}

	interactionId, err := warpy.WriteInteractionToWarpy(
		self.Ctx, self.Config.WarpySyncer, input, self.Log, self.sequencerClient, self.Config.WarpySyncer.WriterApiKey)
	if err != nil {
		self.saveTxToFile(input, true)
		return err
	}

	self.saveTxToFile(input, false)
	self.Log.WithField("interactionId", interactionId).Info("Interaction sent to Warpy")
	self.monitor.GetReport().WarpySyncer.State.WriterInteractionsToWarpy.Inc()
	return
}

// Returns a map of wallet address to a list of discord roles.
// If wallet not registered in warpy the map will not contain the key.
// If wallet found but no roles the map will point to an empty collection.
func (self *Writer) walletAddressToDiscordRoles(payloads *[]InteractionPayload) (walletToRoles *map[string][]string, err error) {
	addresses := make([]string, 0, len(*payloads))
	for _, p := range *payloads {
		addresses = append(addresses, p.FromAddress)
	}
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
			self.Log.WithError(err).
				Warn("Could not process assets sum, retrying...")
			return err
		}).
		Run(func() error {
			result, err := warpy.GetWalletToDiscordIdMap(self.httpClient, self.Config.WarpySyncer.SyncerDreUrl, &addresses, self.Log)
			if err != nil {
				self.Log.WithError(err).Warn("Could not retrieve sender Discord ids")
				return err
			}

			if result.WalletToDiscordId == nil || len(result.WalletToDiscordId) == 0 {
				self.Log.
					WithField("addresses", strings.Join(addresses, ",")).
					Debug("No discord ids found for specified address, exiting")
				return nil
			}

			ids := make([]string, 0, len(result.WalletToDiscordId))
			for _, w := range addresses {
				id := result.WalletToDiscordId[strings.ToLower(w)]
				if len(id) > 15 {
					ids = append(ids, id)
				} else {
					self.Log.WithField("from_address", w).
						Info("Address not registered in Warpy, skipping")
				}
			}

			walletToRoles = &map[string][]string{}
			rolesPayload, err := warpy.GetSendersRoles(self.httpClient, self.Config.WarpySyncer.SyncerWarpyApiUrl, &ids, self.Log)
			if err != nil {
				self.Log.
					WithError(err).Warn("Could not retrieve senders roles")
				return err
			}

			for _, w := range addresses {
				if id, ok := result.WalletToDiscordId[strings.ToLower(w)]; ok {
					(*walletToRoles)[w] = (*rolesPayload).IdToRoles[id]
				}
			}

			return nil
		})
	return
}

func (self Writer) calculateCap() int64 {
	numberOfRewards := (self.Config.WarpySyncer.WriterIntegrationDurationInSec / self.Config.WarpySyncer.BlockDownloaderPollerInterval)
	return self.Config.WarpySyncer.WriterPointsCap / numberOfRewards
}

func (self Writer) saveTxToFile(input Input, withError bool) {
	parsedInput, err := json.Marshal(input)
	if err != nil {
		self.Log.WithError(err).Error("could not parsed input")
		return
	}
	now := time.Now()
	pwd, _ := os.Getwd()
	timeFormatted := now.Format(time.RFC3339)
	if withError {
		err = os.WriteFile(fmt.Sprintf("%s/src/warpy_sync/files/errors/%s.json", pwd, timeFormatted), parsedInput, 0644)
	} else {
		err = os.WriteFile(fmt.Sprintf("%s/src/warpy_sync/files/txs/%s.json", pwd, timeFormatted), parsedInput, 0644)
	}
	if err != nil {
		self.Log.WithError(err).Error("could not parsed input")
		return
	}
}
