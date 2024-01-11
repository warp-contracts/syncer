package redstone_tx_sync

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	sequencer_types "github.com/warp-contracts/syncer/src/utils/sequencer/types"
	"gorm.io/gorm"
)

type Client struct {
	db              *gorm.DB
	Log             *logrus.Entry
	ethClient       *ethclient.Client
	sequencerClient *sequencer.Client
}

func NewClient(config *config.Config) (self *Client) {
	self = new(Client)
	self.Log = logger.NewSublogger("redstone_tx_sync_client")

	return
}

func (self *Client) WithEthClient(url EthUrl) *Client {
	ethClient, err := GetEthClient(self.Log, url)
	if err != nil {
		self.Log.WithError(err).Error("Could not get ETH client")
		return nil
	}
	self.ethClient = ethClient
	return self
}

func (self *Client) WithDB(db *gorm.DB) *Client {
	self.db = db
	return self
}

func (self *Client) WithSequencerClient(sequencerClient *sequencer.Client) *Client {
	self.sequencerClient = sequencerClient
	return self
}

func (self *Client) GetLastSyncedBlockHeight(ctx context.Context) (lastSyncedHeight int64, err error) {
	err = self.db.WithContext(ctx).
		Raw(`SELECT finished_block_height
		FROM sync_state
		WHERE name = ?;`, model.SyncedComponentRedstoneTxSyncer).
		Scan(&lastSyncedHeight).Error

	return
}

func (self *Client) GetCurrentBlockHeight() (currentBlockHeight int64, currentBlockHash string, err error) {
	header, err := self.ethClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return
	}

	currentBlockHeight = header.Number.Int64()
	currentBlockHash = header.TxHash.String()

	return
}

func (self *Client) GetBlockInfo(blockNumber int64) (blockInfo *types.Block, err error) {
	blockInfo, err = self.ethClient.BlockByNumber(context.Background(), big.NewInt(blockNumber))
	if err != nil {
		return
	}
	return
}

func (self *Client) UpdateSyncState(ctx context.Context, blockHeight int64, blockHash string) (err error) {
	err = self.db.WithContext(ctx).
		Exec(`UPDATE sync_state
		SET finished_block_height = ?, finished_block_hash = ?
		WHERE name = ?;`, blockHeight, blockHash, model.SyncedComponentRedstoneTxSyncer).
		Error

	if err != nil {
		return
	}
	return
}

func (self *Client) CheckTxForData(tx *types.Transaction, data string, ctx context.Context) (txContainsData bool) {
	txContainsData = false
	encodedString := hex.EncodeToString(tx.Data())
	if strings.Contains(encodedString, data) {
		txContainsData = true
	}
	return
}

func (self *Client) GetTxSenderHash(tx *types.Transaction) (txSenderHash string, err error) {
	signer := types.LatestSignerForChainID(tx.ChainId())
	sender, err := types.Sender(signer, tx)
	txSenderHash = sender.Hash().String()
	return
}

func (self *Client) WriteInteractionToWarpy(ctx context.Context, tx *types.Transaction, arweaveSigner string, input json.Marshaler, contractId string) (interactionId string, err error) {
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
