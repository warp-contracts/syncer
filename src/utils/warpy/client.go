package warpy

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	sequencer_types "github.com/warp-contracts/syncer/src/utils/sequencer/types"
)

func GetSenderRoles(httpClient *resty.Client, url string, senderDiscordId string, log *logrus.Entry) (roles *[]string, err error) {
	resp, err := httpClient.SetBaseURL(url).R().
		SetResult([]string{}).
		ForceContentType("application/json").
		SetQueryParams(map[string]string{
			"id": senderDiscordId,
		}).
		SetHeader("Accept", "application/json").
		Get("/v1/userRoles")

	if err != nil {
		log.WithError(err).Warn("Could not retrieve sender roles")
		return
	}

	if !resp.IsSuccess() {
		log.WithField("statusCode", resp.StatusCode()).Warn("Sender roles request has not been successful")
		return
	}

	roles, ok := resp.Result().(*[]string)
	if !ok {
		log.Warn("Failed to parse response")
		return
	}
	return
}

func GetWalletToDiscordIdMap(httpClient *resty.Client, url string, addresses *[]string, log *logrus.Entry) (senderIdPayload *model.WalletDiscordIdPayload, err error) {
	if err != nil {
		return
	}

	resp, err := httpClient.SetBaseURL(url).R().
		SetResult(model.WalletDiscordIdPayload{}).
		ForceContentType("application/json").
		SetBody(map[string]interface{}{
			"addresses": *addresses,
		}).
		SetHeader("Accept", "application/json").
		Post("/warpy/user-ids")

	if err != nil {
		return
	}

	if !resp.IsSuccess() {
		log.WithField("statusCode", resp.StatusCode()).WithField("response", resp).
			Warn("Sender Discord id request has not been successful")
		err = errors.New("sender Discord id request has not been successful")
		return
	}

	senderIdPayload, ok := resp.Result().(*model.WalletDiscordIdPayload)
	if !ok {
		log.Warn("Failed to parse response")

		return
	}

	return
}

func WriteInteractionToWarpy(ctx context.Context, arweaveSigner string, input json.Marshaler, contractId string, log *logrus.Entry, sequencerClient *sequencer.Client) (interactionId string, err error) {
	signer, err := bundlr.NewArweaveSigner(arweaveSigner)
	if err != nil {
		log.WithError(err).Error("Could not create Arweave Signer")
		return
	}

	interactionId, err = sequencerClient.UploadInteraction(ctx, input, sequencer_types.WriteInteractionOptions{ContractTxId: contractId}, signer)
	if err != nil {
		log.WithError(err).Error("Could not write interaction to Warpy")
		return
	}
	return
}
