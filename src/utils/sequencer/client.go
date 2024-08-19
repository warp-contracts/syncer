package sequencer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"slices"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/sequencer/requests"
	"github.com/warp-contracts/syncer/src/utils/sequencer/responses"
	"github.com/warp-contracts/syncer/src/utils/sequencer/types"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/warp"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	*BaseClient
}

var (
	MAX_DATA_ITEM_SIZE = 20480
)

func NewClient(config *config.Sequencer) (self *Client) {
	self = new(Client)
	self.BaseClient = newBaseClient(config)
	return
}

func (self *Client) GetNonce(ctx context.Context, signatureType bundlr.SignatureType, owner string) (out *responses.GetNonce, resp *resty.Response, err error) {
	resp, err = self.GetClient().
		R().
		SetContext(ctx).
		SetBody(requests.GetNonce{
			SignatureType: signatureType,
			Owner:         owner,
		}).
		SetResult(&responses.GetNonce{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/json").
		EnableTrace().
		Post("/api/v1/nonce")
	if err != nil {
		return
	}

	// fmt.Printf("The request for nonce took %d ms\n", resp.Request.TraceInfo().TotalTime.Milliseconds())
	out, ok := resp.Result().(*responses.GetNonce)
	if !ok {
		err = ErrFailedToParse
		return
	}
	return
}

func (self *Client) UploadReader(ctx context.Context, reader io.Reader, mode types.BroadcastMode, apiKey string) (out *responses.Upload, resp *resty.Response, err error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	resp, err = self.GetClient().
		R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&responses.Upload{}).
		ForceContentType("application/json").
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("X-Broadcast-Mode", string(mode)).
		SetHeader("X-Api-Key", apiKey).
		EnableTrace().
		Post(self.config.UploadEndpoint)
	if err != nil {
		return
	}

	// fmt.Printf("The request with the data item took %d ms\n", resp.Request.TraceInfo().TotalTime.Milliseconds())
	out, ok := resp.Result().(*responses.Upload)
	if !ok {
		err = ErrFailedToParse
		return
	}

	return
}

func (self *Client) Upload(ctx context.Context, item *bundlr.BundleItem, mode types.BroadcastMode, apiKey string) (out *responses.Upload, resp *resty.Response, err error) {
	reader, err := item.Reader()
	if err != nil {
		return
	}

	return self.UploadReader(ctx, reader, mode, apiKey)
}

func (self *Client) UploadInteraction(ctx context.Context, input json.Marshaler, options types.WriteInteractionOptions, signer bundlr.Signer, apiKey string) (out string, err error) {
	if options.ContractTxId == "" {
		err = errors.New("Contract tx id not passed")
		return
	}

	bundleItem := new(bundlr.BundleItem)

	parsedInput, err := json.Marshal(input)
	if err != nil {
		return
	}

	if len(options.Tags) > 0 {
		bundleItem.Tags = append(bundleItem.Tags, options.Tags...)
	}

	bundleItem.Tags = append(bundleItem.Tags,
		bundlr.Tag{Name: smartweave.TagAppName, Value: smartweave.TagAppNameValue},
		bundlr.Tag{Name: smartweave.TagAppVersion, Value: smartweave.TagAppVersion},
		bundlr.Tag{Name: smartweave.TagSDK, Value: smartweave.TagSDKValue},
		bundlr.Tag{Name: smartweave.TagContractTxId, Value: options.ContractTxId},
		bundlr.Tag{Name: smartweave.TagInputFormat, Value: smartweave.TagInputFormatTagValue},
		bundlr.Tag{Name: smartweave.TagInput, Value: string(parsedInput)},
	)
	if options.Vrf {
		bundleItem.Tags = append(bundleItem.Tags,
			bundlr.Tag{Name: warp.TagRequestVrf, Value: "true"},
		)
	}

	var dataFields types.InteractionDataField
	var data []byte

	tagsBytes, err := bundleItem.Tags.Marshal()
	if err != nil {
		return
	}

	if len(tagsBytes) > 4096 {
		self.log.WithField("taglen", len(tagsBytes)).Debug("Tags len over 4096")
	} else {
		self.log.WithField("taglen", len(tagsBytes)).Debug("Tags len under 4096")
	}

	inputFormatTagIndex := slices.Index(bundleItem.Tags, bundlr.Tag{Name: smartweave.TagInputFormat, Value: smartweave.TagInputFormatTagValue})
	bundleItem.Tags = slices.Delete(bundleItem.Tags, inputFormatTagIndex, inputFormatTagIndex+1)

	bundleItem.Tags = append(bundleItem.Tags, bundlr.Tag{Name: smartweave.TagInputFormat, Value: smartweave.TagInputFormatDataValue})

	inputTagIndex := slices.Index(bundleItem.Tags, bundlr.Tag{Name: smartweave.TagInput, Value: string(parsedInput)})
	bundleItem.Tags = slices.Delete(bundleItem.Tags, inputTagIndex, inputTagIndex+1)

	dataFields.Input = input

	if options.ManifestData != nil {
		dataFields.Manifest = options.ManifestData
	}

	if reflect.DeepEqual(dataFields, types.InteractionDataField{}) {
		bundleItem.Data = arweave.Base64String(fmt.Sprint(rand.Int())[0:4])
	} else {
		data, err = json.Marshal(dataFields)
		if err != nil {
			return
		}
		bundleItem.Data = arweave.Base64String(data)
	}
	bundleItem.SignatureType = options.SignatureType

	err = bundleItem.Sign(signer)
	if err != nil {
		return
	}

	out = bundleItem.Id.Base64()

	if bundleItem.Size() > MAX_DATA_ITEM_SIZE {
		err = errors.New("Interaction data item size exceeds maximum size.")
		return
	}

	_, _, err = self.Upload(ctx, bundleItem, types.BroadcastModeSync, apiKey)
	if err != nil {
		return
	}

	return
}
